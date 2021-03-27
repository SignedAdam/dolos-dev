package main

import (
	"context"
	captchasolver "dolos-dev/pkg/driver/captcha/pysolver"
	"dolos-dev/pkg/helperfuncs"
	"dolos-dev/pkg/structs"
	"dolos-dev/pkg/switcher"
	"fmt"
	"math/rand"
	"sync"
	"time"

	cmdcolor "github.com/TwinProduction/go-color"
)

func (handler *StockAlertHandler) addMetrics(str string, taskID int) string {
	taskIDPrefix := ""
	if taskID > 0 {
		taskIDPrefix = fmt.Sprint(cmdcolor.Bold, cmdcolor.Red, "#", taskID, "://")
	}
	handler.mutex.RLock()
	str = fmt.Sprint(cmdcolor.Bold, cmdcolor.Gray, " [", cmdcolor.Blue, " B: ", handler.metrics.inStockSeen, cmdcolor.Green, " | S: ", handler.metrics.heBorght, cmdcolor.Yellow, " | C: ", handler.metrics.captchaSeen, cmdcolor.Gray, " ] ", cmdcolor.Reset, str)
	handler.mutex.RUnlock()

	str = fmt.Sprint(taskIDPrefix, str)

	return str
}

func (handler *StockAlertHandler) stockChecker(wgSeleniumExit *sync.WaitGroup, ctx context.Context, productURL structs.ProductURL, globalConfig structs.GlobalConfig, taskID int) {

	webshop, webshopKind, err := switcher.GetWebshop(productURL.URL)
	if err != nil {
		helperfuncs.Log(handler.addMetrics("Failed to init webshop interface (%v)", taskID), err)
		wgSeleniumExit.Done()
		return
	}

	var (
		stockCheckInterval  int
		globalProxyLifetime int
		useProxies          bool
		proxyCopies         []structs.Proxy //= structs.Proxy{}
		lastProxySet        time.Time
		proxies             []*structs.Proxy
		proxyLifecycle      bool = false
	)

	switch webshopKind {
	case structs.WEBSHOP_AMAZON, structs.WEBSHOP_AMAZONNL, structs.WEBSHOP_AMAZONIT, structs.WEBSHOP_AMAZONFR, structs.WEBSHOP_AMAZONDE:
		stockCheckInterval = globalConfig.AmazonStockCheckInterval
		globalProxyLifetime = globalConfig.AmazonProxyLifetime
		useProxies = globalConfig.AmazonUseProxies
	}

	if useProxies {
		if globalProxyLifetime > -1 {
			proxyLifecycle = true
		}
		handler.mutex.RLock()
		//find suitable proxies
		proxies, err = helperfuncs.FindNextProxy(productURL.ProxiesCount, nil, handler.Proxies, webshopKind, globalProxyLifetime)
		handler.mutex.RUnlock()
		if err != nil {
			helperfuncs.Log(handler.addMetrics("Failed to get next proxies for %s [URL: %s] (%v)", taskID), productURL.Name, productURL.URL, err)
			wgSeleniumExit.Done()
			return
		}
		handler.mutex.RLock()
		for _, proxy := range proxies {
			proxyCopies = append(proxyCopies, *proxy)
		}
		handler.mutex.RUnlock()

		lastProxySet = time.Now()
	}

	handler.mutex.Lock()
	seleniumSession, err := handler.seleniumHandler.NewSession(&proxyCopies)
	handler.mutex.Unlock()
	if err != nil {
		helperfuncs.Log(handler.addMetrics("Failed to init selenium instance for this task (%v)", taskID), err)
		wgSeleniumExit.Done()
		return
	}

	_ = seleniumSession

	for {
		select {
		case <-ctx.Done():
			//if the sigStopChan channel is signalled then we return from the function to kill the thread
			helperfuncs.Log(handler.addMetrics("Exiting stockChecker thread", taskID))
			/*
				err = seleniumSession.Webdriver.Close()
				if err != nil {
					fmt.Println("failed to close selenium session ", err)
				}
			*/
			err = seleniumSession.Webdriver.Quit()
			if err != nil {
				fmt.Println("failed to quit selenium session ", err)
			}
			wgSeleniumExit.Done()
			return
		default:
			checkStartTime := time.Now()
			inStock, useAddToCartButton, captcha, captchaData, err := seleniumSession.CheckStockStatus(productURL, webshop, globalConfig.DebugScreenshots)
			if err != nil {
				helperfuncs.Log(handler.addMetrics("Failed to check stock for %s [URL: %s] (%v)", taskID), productURL.Name, productURL.URL, err)
			} else {
				if captcha {
					helperfuncs.Log(handler.addMetrics("Captcha found", taskID))

					if captchaData.CaptchaURL == "" {
						helperfuncs.Log(handler.addMetrics("Captcha does not have a URL", taskID))

					} else {
						//put session key and captcha data into CaptchaSolverMap
						handler.mutex.Lock()
						handler.CaptchaSolver[captchaData.SessionID] = captchaData
						handler.metrics.captchaSeen++
						handler.mutex.Unlock()

						captchaToken, err := captchasolver.SolveCaptcha(captchaData.CaptchaURL, globalConfig.CaptchaSolverEndpoint)
						if err != nil {
							helperfuncs.Log(handler.addMetrics("Failed to solve captcha (%v)", taskID), err)
							wgSeleniumExit.Done()
							return
						} else {
							helperfuncs.Log(handler.addMetrics("Captcha solved: %s", taskID), captchaToken)
						}

						err = seleniumSession.SolveCaptcha(captchaToken, webshop)
						if err != nil {
							helperfuncs.Log(handler.addMetrics("Failed to complete captcha (%v)", taskID), err)
							wgSeleniumExit.Done()
							return
						}
					}
				}

				if inStock {
					helperfuncs.Log(handler.addMetrics(fmt.Sprint("Product ", productURL.Name, " is in stock!!!!!"), taskID))

					if !productURL.OnlyCheckStock {
						handler.mutex.RLock()
						err = handler.seleniumHandler.Checkout(useAddToCartButton, webshop, productURL)
						handler.mutex.RUnlock()

						if err != nil {
							helperfuncs.Log(handler.addMetrics("Failed to buy %s (%v)", taskID), productURL.Name, err)
						} else {
							handler.mutex.Lock()
							handler.metrics.heBorght++

							if productURL.MaxPurchases > 0 {
								//find current product in list
								for _, product := range handler.ProductURLs {
									if product.ID == productURL.ID {
										product.CurrentPurchases++
										if product.CurrentPurchases >= product.MaxPurchases {
											helperfuncs.Log(handler.addMetrics("Completed purchase quota for product %s. Stopping task", taskID), productURL.Name)
											handler.mutex.Unlock()
											seleniumSession.Webdriver.Quit()
											wgSeleniumExit.Done()
											return
										}
										break
									}
								}
							}
							handler.mutex.Unlock()

							helperfuncs.Log(handler.addMetrics("============", taskID))
							helperfuncs.Log(handler.addMetrics("HE BORGHT", taskID))
							helperfuncs.Log(handler.addMetrics("HE BORGHT", taskID))
							helperfuncs.Log(handler.addMetrics("HE BORGHT", taskID))
							helperfuncs.Log(handler.addMetrics("HE BORGHT", taskID))
							helperfuncs.Log(handler.addMetrics("HE BORGHT", taskID))
							helperfuncs.Log(handler.addMetrics("============", taskID))
						}
					}

					handler.mutex.Lock()
					handler.metrics.inStockSeen++
					handler.mutex.Unlock()
				} else {
					helperfuncs.Log(handler.addMetrics(fmt.Sprint("Product ", productURL.Name, " sold out"), taskID))
				}
			}

			if useProxies {
				//check if proxies needs to be updated
				if lastProxySet.Add(time.Duration(globalProxyLifetime)*time.Minute).Before(time.Now()) && proxyLifecycle {
					helperfuncs.Log(handler.addMetrics("\n==================================================\nchanging proxies\n", taskID))
					handler.mutex.Lock()
					proxies, err = helperfuncs.FindNextProxy(productURL.ProxiesCount, proxies, handler.Proxies, webshopKind, globalProxyLifetime)
					for _, proxy := range proxies {
						proxyCopies = append(proxyCopies, *proxy)
					}
					handler.mutex.Unlock()
					if err != nil {
						helperfuncs.Log(handler.addMetrics("Failed to get next proxies for %s [URL: %s] (%v)", taskID), productURL.Name, productURL.URL, err)
						wgSeleniumExit.Done()
						return
					}
					helperfuncs.Log(handler.addMetrics("proxies changed\n==================================================\n", taskID))

					handler.mutex.Lock()
					seleniumSession.Webdriver.Quit()
					seleniumSession, err = handler.seleniumHandler.NewSession(&proxyCopies)
					handler.mutex.Unlock()

					lastProxySet = time.Now()
				}
			}

			timeElapsed := time.Now().Sub(checkStartTime)
			if timeElapsed.Milliseconds() < int64(stockCheckInterval) {
				sleepTime := (stockCheckInterval + rand.Intn(globalConfig.AmazonStockCheckIntervalDeviation)) - int(timeElapsed.Milliseconds())
				time.Sleep(time.Duration(sleepTime) * time.Millisecond)
			}

		}
	}
}

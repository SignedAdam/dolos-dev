package main

import (
	captchasolver "dolos-dev/pkg/driver/captcha/pysolver"
	"dolos-dev/pkg/helperfuncs"
	"dolos-dev/pkg/structs"
	"dolos-dev/pkg/switcher"
	"fmt"
	"math/rand"
	"os"
	"time"
	cmdcolor"github.com/TwinProduction/go-color"
)

func (handler *StockAlertHandler) addMetrics(str string) string {
	handler.mutex.RLock()
	str = fmt.Sprint(cmdcolor.Bold, cmdcolor.Gray, "[ B: ", cmdcolor.Blue, handler.metrics.inStockSeen, cmdcolor.Green, " | S: ",  handler.metrics.heBorght, cmdcolor.Yellow,  " | C: ", handler.metrics.captchaSeen, cmdcolor.Gray, " ] ", cmdcolor.Reset,  str)
	handler.mutex.RUnlock()

	return str
}

func (handler *StockAlertHandler) stockChecker(sigStopServerChan chan os.Signal, productURL structs.ProductURL, globalConfig structs.GlobalConfig) {
	webshop, webshopKind, err := switcher.GetWebshop(productURL.URL)
	if err != nil {
		helperfuncs.Log(handler.addMetrics("Failed to init webshop interface (%v)"), err)
	}

	var (
		stockCheckInterval  int
		globalProxyLifetime int
		useProxies          bool
		proxyCopy           structs.Proxy = structs.Proxy{}
		lastProxySet        time.Time
		proxy               *structs.Proxy
		proxyLifecycle      bool = false
	)

	switch webshopKind {
	case structs.WEBSHOP_AMAZON:
		stockCheckInterval = globalConfig.AmazonStockCheckInterval
		globalProxyLifetime = globalConfig.AmazonProxyLifetime
		useProxies = globalConfig.AmazonUseProxies
	}

	if useProxies {
		if globalProxyLifetime > -1 {
			proxyLifecycle = true
		}
		handler.mutex.RLock()
		//find suitable proxy
		proxy, err = helperfuncs.FindNextProxy(nil, handler.Proxies, webshopKind, globalProxyLifetime)
		handler.mutex.RUnlock()
		if err != nil {
			helperfuncs.Log(handler.addMetrics("Failed to get next proxy for %s [URL: %s] (%v)"), productURL.Name, productURL.URL, err)
			return
		}
		proxyCopy = *proxy
		lastProxySet = time.Now()
	}

	for {
		select {
		case <-sigStopServerChan:
			//if the sigStopChan channel is signalled then we return from the function to kill the thread
			helperfuncs.Log(handler.addMetrics("Exiting stockChecker thread"))
			return
		default:

			status, captcha, captchaData, err := webshop.CheckStockStatus(productURL, proxyCopy)
			if err != nil {
				helperfuncs.Log(handler.addMetrics("Failed to check stock for %s [URL: %s] (%v)"), productURL.Name, productURL.URL, err)
			}

			if captcha {
				helperfuncs.Log(handler.addMetrics("Captcha found"))
				
				if captchaData.CaptchaURL == "" {
					helperfuncs.Log(handler.addMetrics("Captcha does not have a URL"))

				} else {
					//put session key and captcha data into CaptchaSolverMap
					handler.mutex.Lock()
					handler.CaptchaSolver[captchaData.SessionID] = captchaData
					handler.metrics.captchaSeen ++
					handler.mutex.Unlock()

					captchaToken, err := captchasolver.SolveCaptcha(captchaData.CaptchaURL, globalConfig.CaptchaSolverEndpoint)
					if err != nil {
						helperfuncs.Log(handler.addMetrics("Failed to solve captcha (%v)"), err)

					} else {
						helperfuncs.Log(handler.addMetrics("Captcha solved: %s"), captchaToken)
					}

					/*
						//open chrome to localhost:3077/api/captchasolver/[session_id]
						helperfuncs.CreateSessionHTML(captchaData.SessionID, captchaData.CaptchaURL)
						path := fmt.Sprintf("captchatemplates/%s.html", captchaData.SessionID)

						err = helperfuncs.OpenInBrowser(path) //captchaData.SessionID
						if err != nil {
							helperfuncs.Log("Failed to open browser (%v)", err)
							return
						}

						//wait for captcha to be solved
						helperfuncs.Log("Waiting for captcha solve...")
						for {
							handler.mutex.RLock()
							solved := handler.CaptchaSolver[captchaData.SessionID].Solved
							handler.mutex.RUnlock()
							if solved {
								helperfuncs.Log("Captcha solved. Continuing...")
								handler.mutex.Lock()
								delete(handler.CaptchaSolver, captchaData.SessionID)

								handler.mutex.Unlock()
								break
							}
							time.Sleep(100 * time.Millisecond)
						}

						_ = captchaData
					*/
				}

			} else {
				if status {
					helperfuncs.Log(handler.addMetrics(fmt.Sprint("Product ", productURL.Name, " is in stock!!!!!")))
					handler.mutex.RLock()
					err = handler.seleniumHandler.Checkout(webshop, productURL)
					handler.mutex.RUnlock()

					handler.mutex.Lock()
					handler.metrics.inStockSeen++
					handler.mutex.Unlock()
					if err != nil {
						helperfuncs.Log(handler.addMetrics("Failed to buy %s (%v)"), productURL.Name, err)
					} else {
						handler.mutex.Lock()
						handler.metrics.heBorght++
						handler.mutex.Unlock()

						helperfuncs.Log(handler.addMetrics("============"))
						helperfuncs.Log(handler.addMetrics("HE BORGHT"))
						helperfuncs.Log(handler.addMetrics("HE BORGHT"))
						helperfuncs.Log(handler.addMetrics("HE BORGHT"))
						helperfuncs.Log(handler.addMetrics("HE BORGHT"))
						helperfuncs.Log(handler.addMetrics("HE BORGHT"))
						helperfuncs.Log(handler.addMetrics("============"))
						
					}
				} else {
					helperfuncs.Log(handler.addMetrics(fmt.Sprint("Product ", productURL.Name, " sold out")))
				}

				if useProxies {
					//check if proxy needs to be updated
					if lastProxySet.Add(time.Duration(globalProxyLifetime)*time.Minute).Before(time.Now()) && proxyLifecycle {
						helperfuncs.Log(handler.addMetrics("\n==================================================\nchanging proxies from: " + proxyCopy.IP))
						handler.mutex.Lock()
						proxy, err = helperfuncs.FindNextProxy(proxy, handler.Proxies, webshopKind, globalProxyLifetime)
						proxyCopy = *proxy
						handler.mutex.Unlock()
						if err != nil {
							helperfuncs.Log(handler.addMetrics("Failed to get next proxy for %s [URL: %s] (%v)"), productURL.Name, productURL.URL, err)
							return
						}
						helperfuncs.Log(handler.addMetrics(" to " + proxyCopy.IP + "\n==================================================\n"))

						lastProxySet = time.Now()
					}
				}

			}

			time.Sleep(time.Duration(stockCheckInterval+rand.Intn(globalConfig.AmazonStockCheckIntervalDeviation)) * time.Millisecond)
		}
	}
}

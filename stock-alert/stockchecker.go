package main

import (
	"dolos-dev/pkg/helperfuncs"
	"dolos-dev/pkg/structs"
	"dolos-dev/pkg/switcher"
	"fmt"
	"math/rand"
	"os"
	"time"
)

func (handler *StockAlertHandler) stockChecker(sigStopServerChan chan os.Signal, productURL structs.ProductURL, globalConfig structs.GlobalConfig) {
	webshop, webshopKind, err := switcher.GetWebshop(productURL.URL)
	if err != nil {
		helperfuncs.Log("Failed to init webshop interface (%v)", err)
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
		proxyCopy = *proxy
		lastProxySet = time.Now()
		handler.mutex.RUnlock()
		if err != nil {
			helperfuncs.Log("Failed to get next proxy for %s [URL: %s] (%v)", productURL.Name, productURL.URL, err)
			return
		}
	}

	for {
		select {
		case <-sigStopServerChan:
			//if the sigStopChan channel is signalled then we return from the function to kill the thread
			helperfuncs.Log("Exiting stockChecker thread")
			return
		default:

			status, captcha, captchaData, err := webshop.CheckStockStatus(productURL, proxyCopy)
			if err != nil {
				helperfuncs.Log("Failed to check stock for %s [URL: %s] (%v)", productURL.Name, productURL.URL, err)
			}

			if captcha {
				helperfuncs.Log("Captcha found")

				if captchaData.CaptchaURL == "" {
					helperfuncs.Log("Captcha does not have a URL")

				} else {
					//put session key and captcha data into CaptchaSolverMap
					handler.mutex.Lock()
					handler.CaptchaSolver[captchaData.SessionID] = captchaData
					handler.mutex.Unlock()

					//open chrome to localhost:3077/api/captchasolver/[session_id]
					helperfuncs.CreateSessionHTML(captchaData.SessionID, captchaData.CaptchaURL)
					path := fmt.Sprintf("C:/Users/Vegeta/go/src/dolos-dev/%s.html", captchaData.SessionID)

					err = helperfuncs.OpenInBrowser(path) //captchaData.SessionID
					if err != nil {
						helperfuncs.Log("Failed to open browser (%v)", err)
						return
					}

					//wait for captcha to be solved
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
				}

			} else {
				if status {
					helperfuncs.Log(fmt.Sprint("Product ", productURL.Name, " is in stock!!!!!"))
					handler.mutex.RLock()
					err = handler.seleniumHandler.Checkout(webshop, productURL)
					handler.mutex.RUnlock()
					if err != nil {
						helperfuncs.Log("Failed to buy %s (%v)", productURL.Name, err)
					} else {
						helperfuncs.Log("============")
						helperfuncs.Log("HE BORGHT")
						helperfuncs.Log("HE BORGHT")
						helperfuncs.Log("HE BORGHT")
						helperfuncs.Log("HE BORGHT")
						helperfuncs.Log("HE BORGHT")
						helperfuncs.Log("============")
					}
				} else {
					helperfuncs.Log(fmt.Sprint("Product ", productURL.Name, " sold out"))
				}

				if useProxies {
					//check if proxy needs to be updated
					if lastProxySet.Add(time.Duration(globalProxyLifetime)*time.Minute).Before(time.Now()) && proxyLifecycle {
						helperfuncs.Log("\n==================================================\nchanging proxies from: " + proxyCopy.IP)
						handler.mutex.Lock()
						proxy, err = helperfuncs.FindNextProxy(proxy, handler.Proxies, webshopKind, globalProxyLifetime)
						proxyCopy = *proxy
						handler.mutex.Unlock()
						if err != nil {
							helperfuncs.Log("Failed to get next proxy for %s [URL: %s] (%v)", productURL.Name, productURL.URL, err)
							return
						}
						helperfuncs.Log(" to " + proxyCopy.IP + "\n==================================================\n")

						lastProxySet = time.Now()
					}
				}

			}

			time.Sleep(time.Duration(stockCheckInterval+rand.Intn(globalConfig.AmazonStockCheckIntervalDeviation)) * time.Millisecond)
		}
	}
}

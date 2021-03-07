package main

import (
	"adam/learn-gitlab/pkg/helperfuncs"
	"adam/learn-gitlab/pkg/structs"
	"adam/learn-gitlab/pkg/switcher"
	"fmt"
	"os"
	"time"
)

func (handler *StockAlertHandler) stockChecker(sigStopServerChan chan os.Signal, productURL structs.ProductURL, stockCheckInterval int) {
	webshop, webshopKind, err := switcher.GetWebshop(productURL.URL)
	if err != nil {
		fmt.Println(fmt.Errorf("Failed to init webshop interface (%v)", err))
	}
	/*
		ProxyIP       string
		ProxyPort     string
		ProxyUser     string
		ProxyPassword string
		UseStart      time.Time
	*/
	handler.mutex.RLock()
	//find suitable proxy
	proxy, err := helperfuncs.FindNextProxy(handler.Proxies, webshopKind)
	handler.mutex.RUnlock()
	if err != nil {
		fmt.Println(fmt.Errorf("Failed to get next proxy for %s [URL: %s] (%v)", productURL.Name, productURL.URL, err))
	}

	for {
		select {
		case <-sigStopServerChan:
			//if the sigStopChan channel is signalled then we return from the function to kill the thread
			fmt.Println("Exiting stockChecker thread")
			return
		default:

			status, captcha, captchaData, err := webshop.CheckStockStatus(productURL, proxy)
			if err != nil {
				fmt.Println(fmt.Errorf("Failed to check stock for %s [URL: %s] (%v)", productURL.Name, productURL.URL, err))
			}

			if captcha {
				fmt.Println("Captcha found")

				if captchaData.CaptchaURL == "" {
					fmt.Println(fmt.Errorf("Captcha does not have a URL"))

				} else {
					//put session key and captcha data into CaptchaSolverMap
					handler.mutex.Lock()
					handler.CaptchaSolver[captchaData.SessionID] = captchaData
					handler.mutex.Unlock()

					//open chrome to localhost:3077/api/captchasolver/[session_id]
					helperfuncs.CreateSessionHTML(captchaData.SessionID, captchaData.CaptchaURL)
					path := fmt.Sprintf("C:/Users/Vegeta/go/src/adam/learn-gitlab/%s.html", captchaData.SessionID)

					err = helperfuncs.OpenInBrowser(path) //captchaData.SessionID
					if err != nil {
						fmt.Println(fmt.Errorf("Failed to open browser (%v)", err))
						return
					}

					//wait for captcha to be solved
					for {
						handler.mutex.RLock()
						solved := handler.CaptchaSolver[captchaData.SessionID].Solved
						handler.mutex.RUnlock()
						if solved {
							fmt.Println("Captcha solved. Continuing...")
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
					fmt.Println("Product ", productURL.Name, " is in stock!!!!!")
				} else {
					fmt.Println("Product ", productURL.Name, " sold out")
				}

				//check if proxy needs to be updated
				//find next proxy
				/*
					handler.mutex.RLock()
					//find suitable proxy
					proxy, err := helperfuncs.FindNextProxy(handler.Proxies, webshopKind)
					handler.mutex.RUnlock()
					if err != nil {
						fmt.Println(fmt.Errorf("Failed to get next proxy for %s [URL: %s] (%v)", productURL.Name, productURL.URL, err))
					}
				*/
			}

			time.Sleep(time.Duration(stockCheckInterval) * time.Millisecond)
		}
	}
}

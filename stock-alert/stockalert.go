package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"adam/learn-gitlab/pkg/helperfuncs"
	"adam/learn-gitlab/pkg/structs"
	"adam/learn-gitlab/pkg/switcher"

	"golang.org/x/net/html"
)

type StockAlertHandler struct {
	ProductURLs   []*structs.ProductURL
	CaptchaSolver map[string]*structs.CaptchaWrapper
	mutex         sync.RWMutex
}

func main() {
	sigStopServerChan := make(chan os.Signal, 1)
	signal.Notify(sigStopServerChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	productURLs, err := helperfuncs.LoadState()
	if err != nil {
		fmt.Println(fmt.Errorf("Failed to read config %v", err.Error()))
	}

	handler := StockAlertHandler{
		ProductURLs:   productURLs,
		CaptchaSolver: make(map[string]*structs.CaptchaWrapper),
	}

	err = helperfuncs.CreateSessionHTML("daddadd", "https://images-na.ssl-images-amazon.com/captcha/qujzzelu/Captcha_sylmyxtxkg.jpg")
	if err != nil {
		fmt.Println(fmt.Errorf("Failed create captcha solver HTML file (%v)", err))
	}

	err = helperfuncs.OpenInBrowser("captchatemplates/daddadd.html") //captchaData.SessionID
	if err != nil {
		fmt.Println(fmt.Errorf("Failed to open browser (%v)", err))
	}

	fmt.Println(fmt.Sprintf("Configurations file loaded. %v product config(s) found.", len(handler.ProductURLs)))

	for _, product := range handler.ProductURLs {
		go handler.stockChecker(sigStopServerChan, *product)
	}

	//Initialize our REST API router & endpoints
	mux := http.NewServeMux()

	//game related routes
	mux.HandleFunc("/api/addproducturl", corsHandler(handler.CreateProductURLHandler))
	mux.HandleFunc("/api/captchasolver", corsHandler(handler.CaptchaSolverHandler))
	mux.HandleFunc("/api/test", corsHandler(handler.Test))

	//serve the API on port 3077
	http.ListenAndServe(":3077", mux)

	log.Println("server stopped")
}

func (handler *StockAlertHandler) stockChecker(sigStopServerChan chan os.Signal, productURL structs.ProductURL) {
	webshop, err := switcher.GetWebshop(productURL.URL)
	if err != nil {
		fmt.Println(fmt.Errorf("Failed to init webshop interface (%v)", err))
	}

	for {
		select {
		case <-sigStopServerChan:
			//if the sigStopChan channel is signalled then we return from the function to kill the thread
			fmt.Println("Exiting stockChecker thread")
			return
		default:

			status, captcha, captchaData, err := webshop.CheckStockStatus(productURL)
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
			}

			time.Sleep(10 * time.Second)
		}
	}
}

func renderNode(n *html.Node) string {
	var buf bytes.Buffer
	w := io.Writer(&buf)
	html.Render(w, n)
	return buf.String()
}

func (handler *StockAlertHandler) CaptchaSolverHandler(w http.ResponseWriter, r *http.Request) {
	captchaChars := r.FormValue("captchachars")
	if captchaChars == "" {
		logAndWriteResponse(w, "Missing captchaChars", http.StatusBadRequest)
		return
	}

	sessionID := r.FormValue("sessionid")
	if sessionID == "" {
		logAndWriteResponse(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	handler.mutex.Lock()
	handler.CaptchaSolver[sessionID].Solved = true
	handler.CaptchaSolver[sessionID].CaptchaChars = captchaChars
	handler.mutex.Unlock()

	logAndWriteResponse(w, "Captcha solved", http.StatusOK)
}

func (handler *StockAlertHandler) Test(w http.ResponseWriter, r *http.Request) {
	body, err := helperfuncs.GetBodyHTML(handler.ProductURLs[0].URL)
	if err != nil {
		logAndWriteResponse(w, "Failed to get html %v", http.StatusBadRequest, err.Error())
		return
	}
	/*
		if url == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, )
			return
		}
	*/

	doc, err := html.Parse(body)
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed to parse body into a html document (%v)", err))
		return
	}
	newBody := renderNode(doc)
	w.Header().Set("Content-Type", "text/html")

	fmt.Fprint(w, newBody)
}

//CreateProductURLHandler handles the creation of a new team
func (handler *StockAlertHandler) CreateProductURLHandler(w http.ResponseWriter, r *http.Request) {
	//decode the request body into a struct representing this product
	productURL := structs.ProductURL{}
	err := json.NewDecoder(r.Body).Decode(&productURL)
	if err != nil {
		logAndWriteResponse(w, "Failed to decode product data from json body %v", http.StatusBadRequest, err.Error())
		return
	}

	handler.ProductURLs = append(handler.ProductURLs, &productURL)
	err = helperfuncs.SaveState(handler.ProductURLs)
	if err != nil {
		logAndWriteResponse(w, "Failed to save product data to file %v", http.StatusBadRequest, err.Error())
		return
	}

	logAndWriteResponse(w, "%s", http.StatusOK)
}

//this gets rid of the hecking cors error 😡 it's used like a bootleg middleware
func corsHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		w.Header().Add("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

		if r.Method == "OPTIONS" {
			http.Error(w, "No Content", http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

//logAndWriteResponse is wrapper for logging that logs to console and also writes to the http writer - just for convenience
func logAndWriteResponse(w http.ResponseWriter, format string, statusCode int, params ...interface{}) {
	if statusCode != http.StatusOK {
		log.Println(fmt.Errorf(format, params...).Error())
	} else {
		fmt.Println(fmt.Sprintf(format, params...))
	}

	w.WriteHeader(statusCode)
	fmt.Fprintf(w, format, params...)
}

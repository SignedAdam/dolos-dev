package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	seleniumdriver "dolos-dev/pkg/driver/selenium"
	"dolos-dev/pkg/helperfuncs"
	"dolos-dev/pkg/structs"
)

//StockAlertHandler represents an instance of this service
type StockAlertHandler struct {
	ProductURLs   []*structs.ProductURL
	CaptchaSolver map[string]*structs.CaptchaWrapper
	mutex         sync.RWMutex

	GlobalConfig *structs.GlobalConfig
	Proxies      []*structs.Proxy

	seleniumHandler *seleniumdriver.SeleniumHandler

	metrics metrics
}

type metrics struct {
	inStockSeen int
	heBorght    int
	captchaSeen int
}

func main() {
	sigStopServerSignal := make(chan os.Signal, 1)
	chanReadyForExit := make(chan bool, 1)
	var wgSeleniumExit sync.WaitGroup
	signal.Notify(sigStopServerSignal, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGQUIT)

	handler := StockAlertHandler{
		CaptchaSolver: make(map[string]*structs.CaptchaWrapper),
	}

	ctxStockChecker, stockCheckerCancel := context.WithCancel(context.Background())
	ctxCheckout, checkoutCancel := context.WithCancel(context.Background())
	go func() {
		<-sigStopServerSignal
		log.Println("Stopping server...")
		time.Sleep(10 * time.Second)
		stockCheckerCancel()
		//wait for all selenium stock checker processes to be terminated
		wgSeleniumExit.Wait()
		fmt.Println("Exiting all checkout processes")
		checkoutCancel()
		handler.mutex.Lock()
		handler.seleniumHandler.CloseAll()
		fmt.Println("Closed all selenium checkout related processes")
		handler.exitHandler()
		handler.mutex.Unlock()
		helperfuncs.KillProcesses("chromedriver.exe", "OpenJDK Platform binary", "Runtime Broker")
		chanReadyForExit <- true
	}()

	err := helperfuncs.LoadAllConfigs(&handler.ProductURLs, &handler.GlobalConfig, &handler.Proxies)
	if err != nil {
		helperfuncs.Log("Failed to read config files (%v)", err.Error())
		return
	}
	helperfuncs.Log("Configuration files loaded: \n\t%v product config(s) found \n\t%v proxies found", len(handler.ProductURLs), len(handler.Proxies))

	seleniumHandler, err := seleniumdriver.Init()
	if err != nil {
		helperfuncs.Log("Failed to init selenium service (%v)", err)
	}
	handler.seleniumHandler = seleniumHandler

	//TODO username and pass from globalconfig
	err = seleniumHandler.CreateCheckoutSessions(handler.GlobalConfig.CheckoutInstancesPerWebshop, ctxCheckout, *handler.GlobalConfig, handler.ProductURLs)
	if err != nil {
		helperfuncs.Log("Failed to start selenium browser instances (%v)", err)
	}

	ij := 0
	for _, product := range handler.ProductURLs {
		for i := 0; i < product.Threads; i++ {
			ij++
			handler.mutex.Lock()
			go handler.stockChecker(&wgSeleniumExit, ctxStockChecker, *product, *handler.GlobalConfig, ij)
			handler.mutex.Unlock()
			wgSeleniumExit.Add(1)
			time.Sleep(5000 * time.Millisecond)
		}
	}

	//Initialize our REST API router & endpoints
	mux := http.NewServeMux()

	//game related routes
	mux.HandleFunc("/api/addproducturl", corsHandler(handler.CreateProductURLHandler))
	mux.HandleFunc("/api/captchasolver", corsHandler(handler.CaptchaSolverHandler))
	mux.HandleFunc("/api/test", corsHandler(handler.Test))

	//serve the API on port 3077
	go http.ListenAndServe(":3077", mux)

	<-chanReadyForExit

	log.Println("Server stopped")
}

func (handler *StockAlertHandler) hekill() {

}

func (handler *StockAlertHandler) exitHandler() {

}

//CaptchaSolverHandler handles http requests for solving captcha
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

//Test huhuehue
func (handler *StockAlertHandler) Test(w http.ResponseWriter, r *http.Request) {
	IPAddress := r.Header.Get("X-Real-Ip")
	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}
	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}

	/*forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded == "" {
		fmt.Fprint(w, "no")
	}*/

	fmt.Println("hello " + IPAddress)
	fmt.Fprint(w, "hello "+IPAddress)
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

//this gets rid of the hecking cors error ???? it's used like a bootleg middleware
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

/*
func test() {
	time.Sleep(3 * time.Second)
	fmt.Println("testing")
	//134.209.29.120:8080
	bodyRead, err := helperfuncs.GetBodyHTML("http://80.201.214.15:3077/api/test", "136.233.215.137", "80", "", "") //"184.82.224.74", "1080"
	if err != nil {
		fmt.Println(fmt.Errorf("Failed to get body %v", err.Error()))
		return
	}

	defer bodyRead.Close()
	bytes, err := ioutil.ReadAll(bodyRead)
	if err != nil {
		fmt.Println(fmt.Errorf("Failed to read body %v", err.Error()))
		return
	}

	fmt.Println(string(bytes))
	fmt.Println("done testing")
	return
}
*/

package selenium

import (
	"time"
	"dolos-dev/pkg/driver/webshop"
	amazonws "dolos-dev/pkg/driver/webshop/amazon"
	"dolos-dev/pkg/structs"
	"fmt"
	"os"
	"sync"

	"github.com/tebeka/selenium"
)

//SeleniumHandler is an instance of this driver that allows for interaction with the selenium interface
type SeleniumHandler struct {
	sessions map[structs.Webshop][]*Session
	sync.RWMutex
}

//Session represents a single selenium webdriver and has a flag 'busy' to indicate whether or not it is being used by another task
type Session struct {
	webdriver selenium.WebDriver
	busy      bool
}

//New creates a new instance of this driver
func New(sessionCount int, sigStopServerChan chan os.Signal, username, password string) (*SeleniumHandler, error) {
	seleniumHandler := &SeleniumHandler{
		sessions: make(map[structs.Webshop][]*Session),
	}
	var wg sync.WaitGroup
	for i := 0; i < sessionCount; i++ {
		wg.Add(1)
		go seleniumHandler.createSession(&wg, i, structs.WEBSHOP_AMAZON, username, password)
	}
	wg.Wait()

	//check if session count is < sessionCount and return error if that is the case
	if len(seleniumHandler.sessions) != sessionCount {
		return nil, fmt.Errorf("Failed to start one or more selenium sessions")
	}

	go seleniumHandler.sessionKeepAlive(sigStopServerChan)

	return seleniumHandler, nil
}

func (handler *SeleniumHandler) sessionKeepAlive(sigStopServerChan chan os.Signal) {
	for {
		select {
		case <-sigStopServerChan:
			fmt.Println("sessionKeepAlive goroutine exiting")
			return
		default:
			handler.Lock()
			for _, webshopSessions:= range handler.sessions {
				for _, session:= range webshopSessions {
					session.webdriver.Refresh()
				}
			}
			handler.Unlock()

			time.Sleep(time.Minute * 15)
		}

	}
}



func (handler *SeleniumHandler) Checkout(webshop webshop.Webshop, product structs.ProductURL) error {

	//refresh page
	//handler.sessions[0].webdriver.Refresh()
	session := handler.getInactiveSession(webshop.GetKind())
	if session == nil {
		return fmt.Errorf("No free sessions available to checkout product %s", product.Name)
	}

	err := webshop.Checkout(product, session.webdriver)
	if err != nil {
		return fmt.Errorf("Failed to checkout product %s (%v)", product.Name, err)
	}

	handler.Lock()
	session.busy = false
	handler.Unlock()
	return nil
}

func (handler *SeleniumHandler) createSession(wg *sync.WaitGroup, id int, webshopKind structs.Webshop, username, password string) {
	defer wg.Done()

	var (
		// These paths will be different on your system.
		seleniumPath     = "selenium/selenium-server/selenium-server-standalone-3.141.59.jar" //client-combined-3.141.59
		chromeDriverPath = "selenium/chrome-driver/chromedriver.exe"
		port             = 8099 + id
	)
	opts := []selenium.ServiceOption{
		//selenium.StartFrameBuffer(),
		selenium.ChromeDriver(chromeDriverPath),
		selenium.Output(os.Stderr),
	}
	_, err := selenium.NewSeleniumService(seleniumPath, port, opts...)
	if err != nil {
		fmt.Println(err)
		return
	}
	//defer service.Stop()

	// Connect to the WebDriver instance running locally.
	caps := selenium.Capabilities{"browserName": "chrome"}
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
	if err != nil {
		fmt.Println(err)
		return
	}

	switch webshopKind {
	case structs.WEBSHOP_AMAZON:
		amazonws.LogInSelenium(username, password, wd)
	}
	newSession := &Session{
		webdriver: wd,
	}
	handler.addSession(webshopKind, newSession)
}

func (handler *SeleniumHandler) addSession(webshopKind structs.Webshop, session *Session) {
	handler.Lock()
	handler.sessions[webshopKind] = append(handler.sessions[webshopKind], session)
	handler.Unlock()
}

func (handler *SeleniumHandler) getInactiveSession(webshopKind structs.Webshop) (session *Session) {
	handler.RLock()
	defer handler.RUnlock()
	for _, session := range handler.sessions[webshopKind] {
		if !session.busy {
			session.busy = true
			return session
		}
	}
	return nil
}

/*
func Test() {
	const (
		// These paths will be different on your system.
		seleniumPath     = "selenium/selenium-server/selenium-server-standalone-3.141.59.jar" //client-combined-3.141.59
		chromeDriverPath = "selenium/chrome-driver/chromedriver.exe"
		port             = 8099
	)
	opts := []selenium.ServiceOption{
		//selenium.StartFrameBuffer(),             // Start an X frame buffer for the browser to run in.
		selenium.ChromeDriver(chromeDriverPath), // Specify the path to GeckoDriver in order to use Firefox.
		selenium.Output(os.Stderr),              // Output debug information to STDERR.
	}
	selenium.SetDebug(true)
	service, err := selenium.NewSeleniumService(seleniumPath, port, opts...)
	if err != nil {
		panic(err) // panic is used only as an example and is not otherwise recommended.
	}
	defer service.Stop()

	// Connect to the WebDriver instance running locally.
	caps := selenium.Capabilities{"browserName": "chrome"}
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
	if err != nil {
		panic(err)
	}
	defer wd.Quit()

	fmt.Println("getting to page")
	// Navigate to the simple playground interface.
	if err := wd.Get("https://www.amazon.com/Crest-White-Toothpaste-Radiant-Mint/dp/B01KZOTRG8/"); err != nil {
		panic(err)
	}

	fmt.Println("clicking")
	// Get a reference to the text box containing code.
	elem, err := wd.FindElement(selenium.ByCSSSelector, "#buy-now-button")
	if err != nil {
		panic(err)
	}

	elem.Click()

	if err := wd.Get("https://www.amazon.com/Crest-White-Toothpaste-Radiant-Mint/dp/B01KZOTRG8/"); err != nil {
		panic(err)
	}

	time.Sleep(20 * time.Second)
}
*/

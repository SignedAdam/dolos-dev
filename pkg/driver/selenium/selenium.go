package selenium

import (
	"dolos-dev/pkg/driver/webshop"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/tebeka/selenium"
)

//SeleniumHandler is an instance of this driver that allows for interaction with the selenium interface
type SeleniumHandler struct {
	sessions []*Session
	sync.RWMutex
}

//Session represents a single selenium webdriver and has a flag 'busy' to indicate whether or not it is being used by another task
type Session struct {
	webdriver selenium.WebDriver
	busy      bool
}

//New creates a new instance of this driver
func New(sessionCount int, webshop *webshop.Webshop, username, password string) (*SeleniumHandler, error) {
	seleniumHandler := &SeleniumHandler{}
	var wg sync.WaitGroup
	for i := 0; i < sessionCount; i++ {
		wg.Add(1)
		go seleniumHandler.createSession(&wg, i, *webshop, username, password)
	}
	wg.Wait()

	//check if session count is < sessionCount and return error if that is the case
	if len(seleniumHandler.sessions) != sessionCount {
		return nil, fmt.Errorf("Failed to start one or more selenium sessions")
	}

	return seleniumHandler, nil
}

func (handler *SeleniumHandler) Checkout(webshop *webshop.Webshop) error {

	//refresh page
	handler.sessions[0].webdriver.Refresh()

	return nil
}

func (handler *SeleniumHandler) createSession(wg *sync.WaitGroup, id int, webshop webshop.Webshop, username, password string) {
	defer wg.Done()

	var (
		// These paths will be different on your system.
		seleniumPath     = "selenium/selenium-server/selenium-server-standalone-3.141.59.jar" //client-combined-3.141.59
		chromeDriverPath = "selenium/chrome-driver/chromedriver.exe"
		port             = 8099 + id
	)
	opts := []selenium.ServiceOption{
		//selenium.StartFrameBuffer(),             // Start an X frame buffer for the browser to run in.
		selenium.ChromeDriver(chromeDriverPath), // Specify the path to GeckoDriver in order to use Firefox.
		selenium.Output(os.Stderr),              // Output debug information to STDERR.
	}
	service, err := selenium.NewSeleniumService(seleniumPath, port, opts...)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer service.Stop()

	// Connect to the WebDriver instance running locally.
	caps := selenium.Capabilities{"browserName": "chrome"}
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
	if err != nil {
		fmt.Println(err)
		return
	}

	newSession := &Session{
		webdriver: wd,
	}

	webshop.LogInSelenium(username, password, wd)

	handler.Lock()
	handler.sessions = append(handler.sessions, newSession)
	handler.Unlock()
}

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

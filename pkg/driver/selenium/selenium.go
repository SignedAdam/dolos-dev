package selenium

import (
	"context"
	"dolos-dev/pkg/driver/webshop"
	amazonws "dolos-dev/pkg/driver/webshop/amazon"
	"dolos-dev/pkg/helperfuncs"
	"dolos-dev/pkg/structs"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

//SeleniumHandler is an instance of this driver that allows for interaction with the selenium interface
type SeleniumHandler struct {
	seleniumService *selenium.Service
	sessions        map[structs.Webshop][]*Session
	//lastPort        int
	sync.RWMutex
}

//Session represents a single selenium webdriver and has a flag 'busy' to indicate whether or not it is being used by another task
type Session struct {
	id        int
	webdriver selenium.WebDriver
	kind      structs.Webshop
	//seleniumService *selenium.Service
	busy bool
}

type SingleSession struct {
	Webdriver selenium.WebDriver
	//SeleniumService *selenium.Service
}

//New creates a new instance of this driver
func (handler *SeleniumHandler) CreateCheckoutSessions(sessionCount int, ctx context.Context, globalConfig structs.GlobalConfig, productURLs []*structs.ProductURL) error {
	//seleniumHandler := &SeleniumHandler{
	handler.sessions = make(map[structs.Webshop][]*Session)
	//	lastPort: 8099,
	//}
	var wg sync.WaitGroup
	for i := 0; i < sessionCount; i++ {
		for _, productURL := range productURLs {
			maxReached, webshopKind, user, pass := handler.maxSessionsCreated(sessionCount, productURL.URL, globalConfig)
			if !maxReached {
				wg.Add(1)
				go handler.createSession(&wg, i, webshopKind, user, pass)
				time.Sleep(5 * time.Second)
			}
		}
	}
	wg.Wait()

	//check if session count is < sessionCount and return error if that is the case
	/*if len(handler.sessions) != sessionCount {
		return fmt.Errorf("Failed to start one or more selenium sessions")
	}*/

	go handler.sessionKeepAlive(ctx, globalConfig)

	return nil
}

func Init() (*SeleniumHandler, error) {
	var (
		// These paths will be different on your system.
		seleniumPath     = "selenium/selenium-server/selenium-server-standalone-3.141.59.jar" //client-combined-3.141.59
		chromeDriverPath = "selenium/chrome-driver/chromedriver.exe"
		port             = 8099
	)

	opts := []selenium.ServiceOption{
		//selenium.StartFrameBuffer(),
		selenium.ChromeDriver(chromeDriverPath),
		//selenium.Output(os.Stderr),

	}
	seleniumSVC, err := selenium.NewSeleniumService(seleniumPath, port, opts...)
	if err != nil {
		return nil, err
	}

	return &SeleniumHandler{seleniumService: seleniumSVC}, nil
}

func (handler *SeleniumHandler) NewSession(proxies *[]structs.Proxy) (*SingleSession, error) {
	/*
		handler.Lock()
		handler.lastPort = handler.lastPort + 1
		port := handler.lastPort
		handler.Unlock()
	*/
	wd, err := createSingleSession( /*port,*/ *proxies)
	if err != nil {
		return nil, err
	}

	return &SingleSession{
		Webdriver: wd,
	}, nil

}

func (handler *SeleniumHandler) createSession(wg *sync.WaitGroup, id int, webshopKind structs.Webshop, username, password string) {
	//we add the session to the handler immediately, so any parent functions will already know it's being worked on
	newSession := &Session{
		id:   id,
		kind: webshopKind,
	}
	handler.addSessionSafe(webshopKind, newSession)

	defer wg.Done()

	webdriver, err := handler.initAndLoginSession(id, webshopKind, username, password)
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed to create selenium session (%v)", err))
		//delete
		handler.deleteBadSession(id, webshopKind)
		return
	}

	handler.Lock()
	newSession.webdriver = webdriver
	handler.Unlock()
}

func createSingleSession( /*port int,*/ proxies []structs.Proxy) (selenium.WebDriver, error) {
	/*var (
		// These paths will be different on your system.
		seleniumPath     = "selenium/selenium-server/selenium-server-standalone-3.141.59.jar" //client-combined-3.141.59
		chromeDriverPath = "selenium/chrome-driver/chromedriver.exe"
	)

	opts := []selenium.ServiceOption{
		//selenium.StartFrameBuffer(),
		selenium.ChromeDriver(chromeDriverPath),
		//selenium.Output(os.Stderr),

	}
	seleniumSVC, err := selenium.NewSeleniumService(seleniumPath, port, opts...)
	if err != nil {
		return nil, nil, err
	}
	//defer service.Stop()
	*/
	caps := selenium.Capabilities{
		"browserName": "chrome",
	}

	//add proxy to this instance's capabilities
	//var pluginPath string
	if len(proxies) > 0 {
		pluginPath, err := createPluginZip(proxies)
		if err != nil {
			return nil, err
		}

		chromeCaps := chrome.Capabilities{
			Path: "",
			Args: []string{
				//"--blink-settings=imagesEnabled=false", // <<<
				"--disable-gpu",
				"--disable-sandbox",
			}}
		chromeCaps.AddExtension(pluginPath)

		caps.AddChrome(chromeCaps)

		err = helperfuncs.DeleteFileOrDir(pluginPath)
		if err != nil {
			return nil, err
		}
	}

	// Connect to the WebDriver instance running locally.
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", 8099))
	if err != nil {
		return nil, err
	}

	return wd, nil
}

func (session *SingleSession) SolveCaptcha(captchaToken string, webshop webshop.Webshop) error {
	err := webshop.SolveCaptcha(session.Webdriver, captchaToken)
	return err
}

func getRandomUserAgent() string {

	userAgents := [10]string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/78.0.3904.108 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.130 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4183.121 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.105 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.116 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.104 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.132 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.141 Safari/537.36",
		"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.88 Safari/537.36",
	}

	return userAgents[rand.Intn(9)]
}

func (session *SingleSession) CheckStockStatus(productURL structs.ProductURL, webshop webshop.Webshop, debugScreenshots bool) (bool, bool, bool, *structs.CaptchaWrapper, error) {

	inStock, inStockCartButton, captcha, captchaSrc, err := webshop.CheckStockStatusSelenium(session.Webdriver, productURL, debugScreenshots)

	captchaStruct := &structs.CaptchaWrapper{
		CaptchaURL: captchaSrc,
	}
	/*
		...
		...
		...
		...
		...
	*/

	return inStock, inStockCartButton, captcha, captchaStruct, err
}

func (handler *SeleniumHandler) Checkout(useAddToCartButton bool, webshop webshop.Webshop, product structs.ProductURL) error {
	//will mark checkout session as not busy
	unbusyFunc := func(session *Session) {
		handler.Lock()
		session.busy = false
		handler.Unlock()
	}

	//refresh page
	//handler.sessions[0].webdriver.Refresh()
	session := handler.getInactiveSession(webshop.GetKind())
	if session == nil {
		return fmt.Errorf("No free sessions available to checkout product %s", product.Name)
	}

	defer unbusyFunc(session)

	err := webshop.CheckoutSidebar(useAddToCartButton, product, session.webdriver)
	if err != nil {
		return fmt.Errorf("Failed to checkout product %s (%v)", product.Name, err)
	}

	return nil
}

func (handler *SeleniumHandler) initAndLoginSession(id int, webshopKind structs.Webshop, username, password string) (selenium.WebDriver, error) {
	wd, err := createSingleSession( /*8099+id, */ nil)
	if err != nil {
		return nil, err
	}

	signInURL, signInFunc := getSignInURLAndFunc(webshopKind)
	err = signInFunc(username, password, wd, signInURL)
	if err != nil {
		err = fmt.Errorf("Failed to log in for this session (%v)", err)
		return nil, err
	}

	return wd, nil
}

func (handler *SeleniumHandler) maxSessionsCreated(maxSessionCount int, url string, globalConfig structs.GlobalConfig) (bool, structs.Webshop, string, string) {
	webshopKind := helperfuncs.GetWebshopFromString(url)

	var username, password string
	switch webshopKind {
	case structs.WEBSHOP_AMAZON, structs.WEBSHOP_AMAZONNL, structs.WEBSHOP_AMAZONFR, structs.WEBSHOP_AMAZONIT, structs.WEBSHOP_AMAZONDE:
		username = globalConfig.AmazonUsername
		password = globalConfig.AmazonPassword
	}

	handler.RLock()
	count := len(handler.sessions[webshopKind])
	handler.RUnlock()
	if count < maxSessionCount {
		return false, webshopKind, username, password
	}

	return true, 0, "", ""
}

//CloseAll closes all webdriver sessions and services related to selenium to prepare for graceful exit of the application
func (handler *SeleniumHandler) CloseAll() {
	handler.Lock()
	for _, sessionList := range handler.sessions {
		for _, session := range sessionList {
			fmt.Println(fmt.Sprintf("Closing selenium instance %v", session.id))
			/*err := session.webdriver.Close()
			if err != nil {
				fmt.Println(err)
			}*/
			err := session.webdriver.Quit()
			if err != nil {
				fmt.Println(err)
			}
			/*err = session.seleniumService.Stop()
			if err != nil {
				fmt.Println(err)
			}*/
		}
	}
	fmt.Println("Closing selenium service")
	err := handler.seleniumService.Stop()
	if err != nil {
		fmt.Println(err)
	}
	handler.Unlock()
}

//delete session that failed to init or whatever
func (handler *SeleniumHandler) deleteBadSession(id int, webshopKind structs.Webshop) {
	handler.Lock()
	for i, session := range handler.sessions[webshopKind] {
		if session.id == id {
			handler.sessions[webshopKind] = remove(handler.sessions[webshopKind], i)
		}
	}
	handler.Unlock()
}

//removes a session from a session slice and returns the result
func remove(s []*Session, i int) []*Session {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

func getSignInURLAndFunc(webshopKind structs.Webshop) (string, func(string, string, selenium.WebDriver, string) error) {
	switch webshopKind {
	case structs.WEBSHOP_AMAZON:
		signInURL := "https://www.amazon.com/ap/signin?openid.pape.max_auth_age=0&openid.return_to=https%3A%2F%2Fwww.amazon.com%2F%3Fref_%3Dnav_signin&openid.identity=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.assoc_handle=usflex&openid.mode=checkid_setup&openid.claimed_id=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.ns=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0&"
		return signInURL, amazonws.LogInSelenium
	case structs.WEBSHOP_AMAZONNL:
		signInURL := "https://www.amazon.nl/ap/signin?openid.pape.max_auth_age=0&openid.return_to=https%3A%2F%2Fwww.amazon.nl%2Fref%3Dnav_ya_signin&openid.identity=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.assoc_handle=nlflex&openid.mode=checkid_setup&openid.claimed_id=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.ns=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0&"
		return signInURL, amazonws.LogInSelenium
	case structs.WEBSHOP_AMAZONDE:
		signInURL := "https://www.amazon.de/ap/signin?openid.pape.max_auth_age=0&openid.return_to=https%3A%2F%2Fwww.amazon.de%2Fref%3Dnav_signin&openid.identity=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.assoc_handle=deflex&openid.mode=checkid_setup&openid.claimed_id=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.ns=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0&"
		return signInURL, amazonws.LogInSelenium
	case structs.WEBSHOP_AMAZONFR:
		signInURL := "https://www.amazon.fr/ap/signin?openid.pape.max_auth_age=0&openid.return_to=https%3A%2F%2Fwww.amazon.fr%2F%3Fref_%3Dnav_custrec_signin&openid.identity=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.assoc_handle=frflex&openid.mode=checkid_setup&openid.claimed_id=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.ns=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0&"
		return signInURL, amazonws.LogInSelenium
	case structs.WEBSHOP_AMAZONIT:
		signInURL := "https://www.amazon.it/ap/signin?openid.pape.max_auth_age=0&openid.return_to=https%3A%2F%2Fwww.amazon.it%2F%3Fref_%3Dnav_custrec_signin&openid.identity=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.assoc_handle=itflex&openid.mode=checkid_setup&openid.claimed_id=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.ns=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0&"
		return signInURL, amazonws.LogInSelenium
	}

	return "", nil
}

func getUserSessionKeepAliveFunc(webshopKind structs.Webshop) func(selenium.WebDriver, structs.GlobalConfig, structs.Webshop) error {
	switch webshopKind {
	case structs.WEBSHOP_AMAZON:
		return amazonws.KeepUserSessionAlive
	case structs.WEBSHOP_AMAZONNL:
		return amazonws.KeepUserSessionAlive
	case structs.WEBSHOP_AMAZONDE:
		return amazonws.KeepUserSessionAlive
	case structs.WEBSHOP_AMAZONFR:
		return amazonws.KeepUserSessionAlive
	case structs.WEBSHOP_AMAZONIT:
		return amazonws.KeepUserSessionAlive
	}

	return nil
}

func (handler *SeleniumHandler) addSessionSafe(webshopKind structs.Webshop, session *Session) {
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

func (handler *SeleniumHandler) sessionKeepAlive(ctx context.Context, globalConfig structs.GlobalConfig) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("sessionKeepAlive goroutine exiting")
			return
		default:
			handler.Lock()
			for _, webshopSessions := range handler.sessions {
				for _, session := range webshopSessions {
					session.webdriver.Refresh()

					userSessionKeepAliveFunc := getUserSessionKeepAliveFunc(session.kind)
					err := userSessionKeepAliveFunc(session.webdriver, globalConfig, session.kind)
					if err != nil {
						fmt.Println(fmt.Errorf("[user session keep alive] Failed to keep user session alive (%v)", err))
					}
				}
			}
			handler.Unlock()

			time.Sleep(time.Minute * 15)
		}
	}
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

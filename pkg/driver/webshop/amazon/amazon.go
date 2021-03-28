package amazon

import (
	"dolos-dev/pkg/helperfuncs"
	"dolos-dev/pkg/structs"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/tebeka/selenium"
	"golang.org/x/net/html"
)

//Webshop represents an instance of this webshop driver
type Webshop struct {
	Kind structs.Webshop
}

//New instantiates a new instance of this driver
func New(webshopKind structs.Webshop) *Webshop {
	return &Webshop{
		Kind: webshopKind,
	}
}

func (shop *Webshop) GetKind() structs.Webshop {
	return shop.Kind
}

//CheckStockStatus checks if a product is in stock on Amazon. Takes a ProductURL struct
//returns:
//bool inStock - boolean representing whether or not the item is in stock
//bool inStockCartButton - boolean representing whether or not the item is in stock, but only with a add to cart button
//bool captcha - boolan representing whether or not a captcha is returned on the page and needs to be solved to proceed
//struct CaptchaWrapper - struct containing all the necessary information about a captcha if one is present
//error - in case something goes wrong in the request
func (shop *Webshop) CheckStockStatus(productURL structs.ProductURL, proxy structs.Proxy) (bool, bool, bool, *structs.CaptchaWrapper, error) {
	body, err := helperfuncs.GetBodyHTML(productURL.URL, proxy.IP, proxy.Port, proxy.User, proxy.Password)
	if err != nil {
		fmt.Println(fmt.Errorf("Failed to get body (%v)", err))
		return false, false, false, nil, err
	}

	bodyOK, inStock, inStockCartButton, captcha, captchaURL := checkStockStatus(body)

	if captcha {
		//generate session id
		sessionID := helperfuncs.GenerateRandomString(6)

		captchaWrapper := &structs.CaptchaWrapper{
			CaptchaURL: captchaURL,
			SessionID:  sessionID,
		}
		return false, false, true, captchaWrapper, nil
	}

	//we check if the body contains an expected element, if it does not, then something went wrong while loading the page
	if !bodyOK && !inStock && !inStockCartButton {
		err = fmt.Errorf("Body failed to properly load for some reason...")

		return false, false, false, nil, err
	}

	return inStock, inStockCartButton, false, nil, nil
}

func getCountryCode(url string) string {
	if strings.Contains(url, "amazon.com") {
		return ".com"
	}
	if strings.Contains(url, "amazon.fr") {
		return ".fr"
	}
	if strings.Contains(url, "amazon.de") {
		return ".de"
	}
	if strings.Contains(url, "amazon.nl") {
		return ".nl"
	}
	if strings.Contains(url, "amazon.it") {
		return ".it"
	}
	return "ERROR"
}

func (shop *Webshop) Checkout(useAddToCartButton bool, product structs.ProductURL, webdriver selenium.WebDriver) error {

	fmt.Println("Attempting to checkout product ", product.Name)
	if err := webdriver.Get(product.URL); err != nil {
		return err
	}

	//find price
	elemPrice, err := webdriver.FindElement(selenium.ByCSSSelector, "#priceblock_ourprice")
	if err != nil {
		return fmt.Errorf("Could not find price element (%v)", err)
	}

	//make sure price is within parameters
	priceString, err := elemPrice.Text()
	if err != nil {
		return fmt.Errorf("Failed get price string from element %s (%v)", priceString, err)
	}

	priceString = strings.ReplaceAll(priceString, "$", "")
	priceString = strings.ReplaceAll(priceString, "€", "")

	price, err := strconv.ParseFloat(priceString, 32)
	if err != nil {
		return fmt.Errorf("Failed to parse price %s (%v)", priceString, err)
	}

	if int(price) > product.MaxPrice || int(price) < product.MinPrice {
		return fmt.Errorf("Price of product %s is outside of parameters (%v)", product.Name, priceString)
	}

	var errContinueBtn error = nil
	if useAddToCartButton {
		//find add to cart button
		elemAddToCartButton, err := webdriver.FindElement(selenium.ByCSSSelector, "#add-to-cart-button")
		if err != nil {
			return fmt.Errorf("Could not find add to cart button element (%v)", err)
		}

		//click add to cart
		err = elemAddToCartButton.Click()
		if err != nil {
			return fmt.Errorf("Failed to click add to cart  button for product %s (%v)", product.Name, err)
		}

		//url to go directly to checkout
		webdriver.Get(fmt.Sprint("https://www.amazon", getCountryCode(product.URL), "/-/en/gp/cart/view.html/ref=lh_co?ie=UTF8&proceedToCheckout.x=129&cartInitiateId=1616029244603&hasWorkingJavascript=1"))
	} else {
		//find buy now button
		elemBuyNowButton, err := webdriver.FindElement(selenium.ByCSSSelector, "#buy-now-button")
		if err != nil {
			return fmt.Errorf("Could not find buy button element (%v)", err)
		}

		//click buy now
		err = elemBuyNowButton.Click()
		if err != nil {
			return fmt.Errorf("Failed to click buy button for product %s (%v)", product.Name, err)
		}

		//find continue button
		elemContinueButton, err := webdriver.FindElement(selenium.ByName, "ppw-widgetEvent:SetPaymentPlanSelectContinueEvent")
		errContinueBtn = err
		if errContinueBtn == nil {
			elemContinueButton.Click()
		}
	}
	//wait ?

	elemPlaceOrder, err := webdriver.FindElement(selenium.ByName, "placeYourOrder1")
	if err != nil {
		if errContinueBtn != nil {
			return fmt.Errorf("Could not find continue OR place order button element (%v)", errContinueBtn)
		}

		return fmt.Errorf("Could not find place order button element (%v)", err)
	}

	elemPlaceOrder.Click()

	return nil
}

func (shop *Webshop) CheckoutSidebar(useAddToCartButton bool, product structs.ProductURL, webdriver selenium.WebDriver) error {

	fmt.Println("Attempting to checkout product ", product.Name)
	if err := webdriver.Get(product.URL + "/ref=olp-opf-redir?aod=1&ie=UTF8&condition=all"); err != nil {
		return err
	}
	/*
		pinnedOffer, err := webdriver.FindElement(selenium.ByID, "aod-pinned-offer")
		if err == nil {
			inStockSidebarPinned, cartButton, _ := checkOffer(webdriver, product, pinnedOffer)
			if inStockSidebarPinned {

				err := checkout(webdriver, product, *cartButton)
				if err != nil {
					return err
				}
			}
		}
	*/
	//get offer list
	offerList, err := webdriver.FindElement(selenium.ByID, "aod-offer-list")
	if err != nil {
		return err
	}

	//loop through div id [aod-offer] elements
	offers, err := offerList.FindElements(selenium.ByID, "aod-offer")
	if err != nil {
		return err
	}

	if len(offers) == 0 {
		return fmt.Errorf("No stock found")
	}

	var offerError error
	for _, offer := range offers {

		inStock, addToCartButton, err := checkOffer(webdriver, product, offer)
		if err != nil {
			offerError = err
			continue
		}
		if inStock {
			err = checkout(webdriver, product, *addToCartButton)
			if err != nil {
				return err
			}
		}
	}

	if offerError != nil {
		return offerError
	}

	return nil
}

func (shop *Webshop) CheckStockStatusSelenium(webdriver selenium.WebDriver, productURL structs.ProductURL, debugScreenshots bool) (bool, bool, bool, string, error) {
	err := webdriver.Get(productURL.URL + "/ref=olp-opf-redir?aod=1&ie=UTF8&condition=all")
	if err != nil {
		return false, false, false, "", err
	}

	_, err = webdriver.FindElement(selenium.ByCSSSelector, "#productTitle")
	if err != nil {
		//couldn't find product title. Maybe captcha?
		images, err := webdriver.FindElements(selenium.ByTagName, "#img")
		if err != nil {
			err = fmt.Errorf("Page not correctly loaded (%v)", err)
			return false, false, false, "", err
		}
		for _, img := range images {
			imgSrc, err := img.GetAttribute("src")
			if err != nil {
				err = fmt.Errorf("img has no src attribute (%v)", err)

			} else {
				if strings.Contains(imgSrc, "captcha") {
					return false, false, true, imgSrc, err
				}

			}
		}
		if debugScreenshots {
			screenshot, screenshotErr := webdriver.Screenshot()
			if screenshotErr != nil {
				fmt.Println("Failed to screenshot")
			}
			imagePath, screenshotErr := helperfuncs.SaveImage(productURL.Name, screenshot)
			if screenshotErr != nil {
				fmt.Println("Failed to save screenshot")
			}
			return false, false, false, "", fmt.Errorf("Page not correctly loaded or something. Screenshot saved under %s (%v)", imagePath, err)
		}

		return false, false, false, "", fmt.Errorf("Page not correctly loaded or something (%v)", err)
	}

	/* //we only check sidebar, not the main page
	_, err = webdriver.FindElement(selenium.ByID, "buy-now-button")
	if err == nil {
		return true, false, false, "", nil
	}

	_, err = webdriver.FindElement(selenium.ByID, "add-to-cart-button")
	if err == nil {
		return true, true, false, "", nil
	}
	*/

	inStockCart, err := shop.CheckStockSidebar(webdriver, productURL, debugScreenshots)

	//we return the same var twice here since if its in stock in the side bar, it will always be as add-to-cart
	return inStockCart, inStockCart, false, "", err
}

func checkOffer(webdriver selenium.WebDriver, productURL structs.ProductURL, parentElement selenium.WebElement) (bool, *selenium.WebElement, error) {
	pinnedOfferPrice, err := parentElement.FindElement(selenium.ByCSSSelector, ".a-price-whole")
	if err != nil {
		return false, nil, err
	}

	//make sure price is within parameters
	priceString, err := pinnedOfferPrice.Text()
	if err != nil {
		return false, nil, err
	}
	/*
		priceString = strings.ReplaceAll(priceString, "$", "")
		priceString = strings.ReplaceAll(priceString, "€", "")
	*/
	price, err := strconv.ParseFloat(priceString, 32)
	if err != nil {
		return false, nil, err
	}

	if int(price) > productURL.MaxPrice || int(price) < productURL.MinPrice {
		return false, nil, err
	}

	addToCartButton, err := parentElement.FindElement(selenium.ByName, "submit.addToCart")
	if err != nil {
		return false, nil, err
	}

	return true, &addToCartButton, nil
}

func (shop *Webshop) CheckStockSidebar(webdriver selenium.WebDriver, productURL structs.ProductURL, debugScreenshots bool) (bool, error) {

	err := webdriver.WaitWithTimeoutAndInterval(func(wd selenium.WebDriver) (bool, error) {
		//for {
		pinnedOffer, err := webdriver.FindElement(selenium.ByCSSSelector, "#all-offers-display-scroller")
		if err != nil {
			return false, nil //fmt.Errorf("Could not find password element (%v)", err)
		}
		if pinnedOffer != nil {
			return true, nil
		}
		return false, nil
		//}
	}, 5*time.Second, 10*time.Millisecond)

	if err != nil {
		if debugScreenshots {
			screenshot, screenshotErr := webdriver.Screenshot()
			if screenshotErr != nil {
				fmt.Println("Failed to screenshot")
			}
			imagePath, screenshotErr := helperfuncs.SaveImage(productURL.Name, screenshot)
			if screenshotErr != nil {
				fmt.Println("Failed to save screenshot")
			}
			return false, fmt.Errorf("timed out looking for all-offers-display-scroller element. Screenshot saved under %s (%v)", imagePath, err)
		}
		return false, fmt.Errorf("timed out looking for all-offers-display-scroller element (%v)", err)
	}

	err = webdriver.WaitWithTimeoutAndInterval(func(wd selenium.WebDriver) (bool, error) {
		//for {
		pinnedOffer, err := webdriver.FindElement(selenium.ByCSSSelector, "#aod-pinned-offer")
		if err != nil {
			return false, nil //fmt.Errorf("Could not find password element (%v)", err)
		}
		if pinnedOffer != nil {
			return true, nil
		}
		return false, nil
		//}
	}, 5*time.Second, 10*time.Millisecond)
	if err != nil {

		if debugScreenshots {
			screenshot, screenshotErr := webdriver.Screenshot()
			if screenshotErr != nil {
				fmt.Println("Failed to screenshot")
			}
			imagePath, screenshotErr := helperfuncs.SaveImage(productURL.Name, screenshot)
			if screenshotErr != nil {
				fmt.Println("Failed to save screenshot")
			}
			return false, fmt.Errorf("timed out looking for all-offers-display-scroller element. Screenshot saved under %s (%v)", imagePath, err)
		}
		return false, fmt.Errorf("timed out looking for aod-pinned-offer element (%v)", err)
	}

	pinnedOffer, err := webdriver.FindElement(selenium.ByCSSSelector, "#aod-pinned-offer")
	if err == nil {
		inStockSidebarPinned, _, _ := checkOffer(webdriver, productURL, pinnedOffer)
		if inStockSidebarPinned {
			return true, nil
		}
	}

	//find div element containing all products
	//div id aod-offer-list
	offerList, err := webdriver.FindElement(selenium.ByCSSSelector, "#aod-offer-list")
	if err != nil {
		return false, fmt.Errorf("Could not find sidebar offer list (%v)", err)
	}

	//loop through div id [aod-offer] elements
	offers, err := offerList.FindElements(selenium.ByCSSSelector, "#aod-offer")
	if err != nil {
		return false, err
	}

	if len(offers) == 0 {
		return false, nil
	}

	var offerError error
	for _, offer := range offers {

		inStock, _, err := checkOffer(webdriver, productURL, offer)
		if inStock {
			return true, nil
		}

		if err != nil {
			offerError = err
		}
	}

	if offerError != nil {
		return false, err
	}

	return false, nil
}

func checkout(webdriver selenium.WebDriver, product structs.ProductURL, addToCartButton selenium.WebElement) error {
	var errContinueBtn error = nil
	//click add to cart
	err := addToCartButton.Click()
	if err != nil {
		return fmt.Errorf("Failed to click add to cart  button for product %s (%v)", product.Name, err)
	}

	//url to go directly to checkout
	webdriver.Get(fmt.Sprint("https://www.amazon", getCountryCode(product.URL), "/-/en/gp/cart/view.html/ref=lh_co?ie=UTF8&proceedToCheckout.x=129&cartInitiateId=1616029244603&hasWorkingJavascript=1"))

	//wait ?

	elemPlaceOrder, err := webdriver.FindElement(selenium.ByName, "placeYourOrder1")
	if err != nil {
		if errContinueBtn != nil {
			return fmt.Errorf("Could not find continue OR place order button element (%v)", errContinueBtn)
		}

		return fmt.Errorf("Could not find place order button element (%v)", err)
	}

	elemPlaceOrder.Click()

	return nil
}

//LogInSelenium logs in to Amazon with the given username & password using the given webdriver interface
func LogInSelenium(username, password string, webdriver selenium.WebDriver, signInURL string) error {

	//navigate to sign in page
	fmt.Println("Signing in")
	//https://www.amazon.nl /ap/signin?openid.pape.max_auth_age=0                                                                     &openid.identity=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.assoc_handle=nlflex&openid.mode=checkid_setup&openid.claimed_id=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.ns=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0&
	//signInURL := "https://www.amazon.com/ap/signin?openid.pape.max_auth_age=0&openid.return_to=https%3A%2F%2Fwww.amazon.com%2F%3Fref_%3Dnav_signin&openid.identity=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.assoc_handle=usflex&openid.mode=checkid_setup&openid.claimed_id=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.ns=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0&"
	if err := webdriver.Get(signInURL); err != nil {
		return err
	}

	//find email text box
	elemEmail, err := webdriver.FindElement(selenium.ByCSSSelector, "#ap_email")
	if err != nil {
		return fmt.Errorf("Could not find email element (%v)", err)
	}

	//fill email
	err = elemEmail.SendKeys(username)

	//find continue button
	elemContinue, err := webdriver.FindElement(selenium.ByCSSSelector, "#continue")
	if err != nil {
		return fmt.Errorf("Could not find login continue element (%v)", err)
	}
	elemContinue.Click()

	webdriver.Wait(func(wd selenium.WebDriver) (bool, error) {
		//for {
		elemPassword, err := webdriver.FindElement(selenium.ByCSSSelector, "#ap_password")
		if err != nil {
			return false, nil
		}
		if elemPassword != nil {
			return true, nil
		}
		return false, nil
		//}
	})

	//find password textbox
	elemPassword, err := webdriver.FindElement(selenium.ByCSSSelector, "#ap_password")
	if err != nil {
		return fmt.Errorf("Could not find password element (%v)", err)
	}

	//fill password textbox
	err = elemPassword.SendKeys(password)

	//find "keep me signed in" button
	elemRememberMe, err := webdriver.FindElement(selenium.ByName, "rememberMe")
	if err != nil {
		return fmt.Errorf("Could not find remember me checkbox element (%v)", err)
	}

	//check checkbox
	err = elemRememberMe.Click()
	if err != nil {
		return fmt.Errorf("Could not click remember me checkbox element (%v)", err)
	}

	//click sign in button
	elemSignIn, err := webdriver.FindElement(selenium.ByCSSSelector, "#signInSubmit")
	if err != nil {
		return fmt.Errorf("Could not find sign in button element (%v)", err)
	}
	elemSignIn.Click()

	//WAIT FOR search field to appear

	webdriver.Wait(func(wd selenium.WebDriver) (bool, error) {
		//for {
		elemSearchBox, err := webdriver.FindElement(selenium.ByID, "twotabsearchtextbox")
		if err != nil {
			return false, nil
		}
		if elemSearchBox != nil {
			return true, nil
		}
		return false, nil
		//}
	})
	return nil
}

func (shop *Webshop) SolveCaptcha(webdriver selenium.WebDriver, captchaToken string) error {

	//solve captchaaaa
	captchaTextBox, err := webdriver.FindElement(selenium.ByCSSSelector, "#captchacharacters")
	if err != nil {
		return fmt.Errorf("Failed to find captcha text box element (%v)", err)
	}

	captchaTextBox.SendKeys(captchaToken)

	continueButton, err := webdriver.FindElement(selenium.ByCSSSelector, "#a-button-text")
	if err != nil {
		return fmt.Errorf("Failed to find continue button element (%v)", err)
	}

	continueButton.Click()

	return nil
}

func checkStockStatus(body io.ReadCloser) (bool, bool, bool, bool, string) {
	var findElement func(bool, *html.Node) (bool, bool, bool, bool, string)
	findElement = func(bodyAlreadyFound bool, n *html.Node) (bool, bool, bool, bool, string) {
		bodyOK := bodyAlreadyFound
		if n.Type == html.ElementNode {
			if n.Data == "span" && !bodyOK {
				for _, a := range n.Attr {
					if a.Key == "id" {
						if a.Val == "productTitle" {
							bodyOK = true
						}
					}
				}
			}

			if n.Data == "input" {
				for _, a := range n.Attr {
					if a.Key == "id" {
						if a.Val == "captchacharacters" {
							fmt.Println("captcha detected")

							//captchaURL = findCaptchaURL(captchaBodyCopy)
							//return false, true, captchaURL
						}

						if a.Val == "buy-now-button" {
							return bodyOK, true, false, false, ""
						}

						if a.Val == "add-to-cart-button" {
							return bodyOK, true, true, false, ""
						}

					}
				}
			}
			if n.Data == "img" {
				for _, a := range n.Attr {
					if a.Key == "src" {
						if strings.Contains(a.Val, "captcha") {
							fmt.Println("Captcha img found")
							return bodyOK, false, false, true, a.Val

						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			newBodyOK, inStock, inStockCartButton, captcha, captchaURL := findElement(bodyOK, c)
			bodyOK = newBodyOK
			if inStock || inStockCartButton || captcha {
				return bodyOK, inStock, inStockCartButton, captcha, captchaURL
			}

		}
		return bodyOK, false, false, false, ""
	}

	doc, err := html.Parse(body)
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed to parse body into a html document (%v)", err))
		return false, false, false, false, ""

	}

	return findElement(false, doc)

}

//obsolete
func findCaptchaURL(body io.ReadCloser) string {
	var findElement func(*html.Node) string
	findElement = func(n *html.Node) string {
		if n.Type == html.ElementNode {

			fmt.Print(n.Data)
			if n.Data == "img" {
				for _, a := range n.Attr {
					if a.Key == "src" {
						if strings.Contains(a.Val, "captcha") {
							return a.Val
						}
					}
				}
			}
		}

		for {
			c := n.FirstChild

			if c == nil {
				c = n.NextSibling
			}
			if c == nil {
				break
			}

			captchaURL := findElement(c)
			if captchaURL != "" {
				return captchaURL
			}
		}
		/*
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				captchaURL := findElement(c)
				if captchaURL != "" {
					return captchaURL
				}

			}
		*/
		return ""
	}

	doc, err := html.Parse(body)
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed to parse body into a html document (%v)", err))
		return ""
	}
	return findElement(doc)
}

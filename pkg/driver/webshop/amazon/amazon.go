package amazon

import (
	"dolos-dev/pkg/helperfuncs"
	"dolos-dev/pkg/structs"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/tebeka/selenium"
	"golang.org/x/net/html"
)

//Webshop represents an instance of this webshop driver
type Webshop struct {
	Kind structs.Webshop
}

//New instantiates a new instance of this driver
func New() *Webshop {
	return &Webshop{
		Kind: structs.WEBSHOP_AMAZON,
	}
}

func (shop *Webshop) GetKind() structs.Webshop {
	return shop.Kind
}

//CheckStockStatus checks if a product is in stock on Amazon. Takes a ProductURL struct
//returns:
//bool inStock - boolean representing whether or not the item is in stock
//bool captcha - boolan representing whether or not a captcha is returned on the page and needs to be solved to proceed
//struct CaptchaWrapper - struct containing all the necessary information about a captcha if one is present
//error - in case something goes wrong in the request
func (shop *Webshop) CheckStockStatus(productURL structs.ProductURL, proxy structs.Proxy) (bool, bool, *structs.CaptchaWrapper, error) {
	body, err := helperfuncs.GetBodyHTML(productURL.URL, proxy.IP, proxy.Port, proxy.User, proxy.Password)
	if err != nil {
		fmt.Println(fmt.Errorf("Failed to get body (%v)", err))
		return false, false, nil, err
	}

	inStock, captcha, captchaURL := checkStockStatus(body)

	if captcha {
		//generate session id
		sessionID := helperfuncs.GenerateRandomString(6)

		captchaWrapper := &structs.CaptchaWrapper{
			CaptchaURL: captchaURL,
			SessionID:  sessionID,
		}
		return false, true, captchaWrapper, nil
	}
	return inStock, false, nil, nil
}

func (shop *Webshop) Checkout(product structs.ProductURL, webdriver selenium.WebDriver) error {

	fmt.Println("Attempting to checkout product ", product.Name)
	if err := webdriver.Get(product.URL); err != nil {
		return err
	}

	//find buy now button
	elemBuyNowButton, err := webdriver.FindElement(selenium.ByCSSSelector, "#buy-now-button")
	if err != nil {
		return fmt.Errorf("Could not find buy button element (%v)", err)
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

	price, err := strconv.ParseFloat(priceString, 32)
	if err != nil {
		return fmt.Errorf("Failed to parce price %s (%v)", priceString, err)
	}

	if int(price) > product.MaxPrice || int(price) < product.MinPrice {
		return fmt.Errorf("Price of product %s is outside of parameters (%v)", product.Name, priceString)
	}

	//click buy now
	err = elemBuyNowButton.Click()
	if err != nil {
		return fmt.Errorf("Failed to click button for product %s (%v)", product.Name, err)
	}

	//wait ?

	//find continue button
	elemContinueButton, errContinueBtn := webdriver.FindElement(selenium.ByName, "ppw-widgetEvent:SetPaymentPlanSelectContinueEvent")
	if errContinueBtn == nil {
		elemContinueButton.Click()
	}

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
func LogInSelenium(username, password string, webdriver selenium.WebDriver) error {

	//navigate to sign in page
	fmt.Println("Signing in")
	signInURL := "https://www.amazon.com/ap/signin?openid.pape.max_auth_age=0&openid.return_to=https%3A%2F%2Fwww.amazon.com%2F%3Fref_%3Dnav_signin&openid.identity=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.assoc_handle=usflex&openid.mode=checkid_setup&openid.claimed_id=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0%2Fidentifier_select&openid.ns=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0&"
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
			return false, fmt.Errorf("Could not find password element (%v)", err)
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

	//click sign in button
	elemSignIn, err := webdriver.FindElement(selenium.ByCSSSelector, "#signInSubmit")
	if err != nil {
		return fmt.Errorf("Could not find sign in button element (%v)", err)
	}
	elemSignIn.Click()

	return nil
}

func checkStockStatus(body io.ReadCloser) (bool, bool, string) {
	var findElement func(*html.Node) (bool, bool, string)
	findElement = func(n *html.Node) (bool, bool, string) {
		if n.Type == html.ElementNode {
			if n.Data == "input" {
				for _, a := range n.Attr {
					if a.Key == "id" {
						if a.Val == "captchacharacters" {
							fmt.Println("captcha detected")

							//captchaURL = findCaptchaURL(captchaBodyCopy)
							//return false, true, captchaURL
						}

						if a.Val == "buy-now-button" {
							return true, false, ""
						}
					}
				}
			}
			if n.Data == "img" {
				for _, a := range n.Attr {
					if a.Key == "src" {
						if strings.Contains(a.Val, "captcha") {
							fmt.Println("Captcha img found")
							return false, true, a.Val

						}
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			inStock, captcha, captchaURL := findElement(c)
			if inStock || captcha {
				return inStock, captcha, captchaURL
			}

		}
		return false, false, ""
	}

	doc, err := html.Parse(body)
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed to parse body into a html document (%v)", err))
		return false, false, ""

	}

	return findElement(doc)

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

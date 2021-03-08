package amazon

import (
	"adam/learn-gitlab/pkg/helperfuncs"
	"adam/learn-gitlab/pkg/structs"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

//Webshop represents an instance of this webshop driver
type Webshop struct {
}

//New instantiates a new instance of this driver
func New() *Webshop {
	return &Webshop{}
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

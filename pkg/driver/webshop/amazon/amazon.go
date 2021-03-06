package amazon

import (
	"adam/learn-gitlab/pkg/helperfuncs"
	"adam/learn-gitlab/pkg/structs"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

type Webshop struct {
}

func New() *Webshop {
	return &Webshop{}
}

func (shop *Webshop) CheckStockStatus(productURL structs.ProductURL) (bool, bool, *structs.CaptchaWrapper, error) {
	body, err := helperfuncs.GetBodyHTML(productURL.URL, "", "", "", "")
	if err != nil {
		fmt.Println(fmt.Errorf("Failed to get body (%v)", err))
	}

	inStock, captcha, captchaURL := checkStockStatus(body)

	if captcha {
		//generate session id
		sessionID := "asbsdf"

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

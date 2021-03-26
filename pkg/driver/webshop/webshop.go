package webshop

import (
	"dolos-dev/pkg/structs"

	"github.com/tebeka/selenium"
)

type Webshop interface {
	CheckStockStatus(structs.ProductURL, structs.Proxy) (bool, bool, bool, *structs.CaptchaWrapper, error)
	CheckStockStatusSelenium(selenium.WebDriver, structs.ProductURL /*, structs.Proxy*/) (bool, bool, bool, string, error)
	SolveCaptcha(selenium.WebDriver, string) error

	GetKind() structs.Webshop
	//LogInSelenium(string, string, selenium.WebDriver) error
	Checkout(bool, structs.ProductURL, selenium.WebDriver) error
	CheckoutSidebar(bool, structs.ProductURL, selenium.WebDriver) error
}

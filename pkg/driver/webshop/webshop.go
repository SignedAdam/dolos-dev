package webshop

import (
	"dolos-dev/pkg/structs"

	"github.com/tebeka/selenium"
)

type Webshop interface {
	CheckStockStatus(structs.ProductURL, structs.Proxy) (bool, bool, *structs.CaptchaWrapper, error)
	GetKind() structs.Webshop
	//LogInSelenium(string, string, selenium.WebDriver) error
	Checkout(structs.ProductURL, selenium.WebDriver) error
}

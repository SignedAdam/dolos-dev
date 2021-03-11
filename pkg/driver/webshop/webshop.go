package webshop

import (
	"dolos-dev/pkg/structs"

	"github.com/tebeka/selenium"
)

type Webshop interface {
	CheckStockStatus(structs.ProductURL, structs.Proxy) (bool, bool, *structs.CaptchaWrapper, error)
	LogInSelenium(string, string, selenium.WebDriver) error
}

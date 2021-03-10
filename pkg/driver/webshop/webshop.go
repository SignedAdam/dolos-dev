package webshop

import (
	"dolos-dev/pkg/structs"
)

type Webshop interface {
	CheckStockStatus(structs.ProductURL, structs.Proxy) (bool, bool, *structs.CaptchaWrapper, error)
}

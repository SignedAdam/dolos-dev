package webshop

import (
	"adam/learn-gitlab/pkg/structs"
)

type Webshop interface {
	CheckStockStatus(structs.ProductURL, structs.Proxy) (bool, bool, *structs.CaptchaWrapper, error)
}

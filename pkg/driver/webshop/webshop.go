package webshop

import (
	"adam/learn-gitlab/pkg/structs"
)

type Webshop interface {
	CheckStockStatus(structs.ProductURL) (bool, bool, *structs.CaptchaWrapper, error)
}

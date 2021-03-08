package structs

import "time"

//ProductURL represents a single product and the necessary data to check its stock
type ProductURL struct {
	Name     string
	URL      string
	MinPrice int `json:"min_price"`
	MaxPrice int `json:"max_price"`
	Threads  int
}

type CaptchaWrapper struct {
	SessionID    string
	CaptchaURL   string
	Solved       bool
	CaptchaChars string
}

type GlobalConfig struct {
	AmazonStockCheckInterval          int `json:"amazon_stock_check_interval"`
	AmazonStockCheckIntervalDeviation int `json:"amazon_stock_check_interval_deviation"`
	AmazonProxyLifetime               int `json:"amazon_proxy_lifetime"`
}

type Proxy struct {
	IP             string
	Port           string
	User           string
	Password       string
	InUse          bool
	LastUsedAmazon time.Time
}

type Webshop int

const (
	WEBSHOP_NONE   Webshop = 0
	WEBSHOP_AMAZON Webshop = 1
)

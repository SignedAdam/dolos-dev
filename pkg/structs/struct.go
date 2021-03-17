package structs

import "time"

//ProductURL represents a single product and the necessary data to check its stock
type ProductURL struct {
	ID 		 int `json:"id"`
	Name     string
	URL      string
	MinPrice int `json:"min_price"`
	MaxPrice int `json:"max_price"`
	Threads  int
	MaxPurchases int `json:"max_purchases"`
	CurrentPurchases int
}

type CaptchaWrapper struct {
	SessionID    string
	CaptchaURL   string
	Solved       bool
	CaptchaChars string
}

type GlobalConfig struct {
	AmazonStockCheckInterval          int    `json:"amazon_stock_check_interval"`
	AmazonStockCheckIntervalDeviation int    `json:"amazon_stock_check_interval_deviation"`
	AmazonUseProxies                  bool   `json:"amazon_use_proxies"`
	AmazonProxyLifetime               int    `json:"amazon_proxy_lifetime"`
	AmazonUsername 					  string `json:"amazon_username"`
	AmazonPassword 					  string `json:"amazon_password"`
	CaptchaSolverEndpoint             string `json:"captcha_solver_endpoint"`
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
	WEBSHOP_AMAZONNL Webshop = 2
	WEBSHOP_AMAZONDE Webshop = 3
	WEBSHOP_AMAZONIT Webshop = 4
	WEBSHOP_AMAZONFR Webshop = 5
)

package structs

//ProductURL represents a single product and the necessary data to check its stock
type ProductURL struct {
	Name     string
	URL      string
	MinPrice int `json:"min_price"`
	MaxPrice int `json:"max_price"`
}

type CaptchaWrapper struct {
	SessionID    string
	CaptchaURL   string
	Solved       bool
	CaptchaChars string
}

type GlobalConfig struct {
	StockCheckInterval int `json:"stock_check_interval"`
}

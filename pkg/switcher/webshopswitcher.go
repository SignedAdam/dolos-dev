package switcher

import (
	"adam/learn-gitlab/pkg/driver/webshop"
	"adam/learn-gitlab/pkg/driver/webshop/amazon"
	"fmt"
	"strings"
)

func GetWebshop(URL string) (webshop.Webshop, error) {
	if strings.Contains(URL, "amazon") {
		return amazon.New(), nil
	}

	return nil, fmt.Errorf("No webshop interface exists for this website")

}

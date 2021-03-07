package switcher

import (
	"adam/learn-gitlab/pkg/driver/webshop"
	"adam/learn-gitlab/pkg/driver/webshop/amazon"
	"adam/learn-gitlab/pkg/structs"
	"fmt"
	"strings"
)

//GetWebshop is a switcher function that gets the correct webshop driver using the given URL
func GetWebshop(URL string) (webshop.Webshop, structs.Webshop, error) {
	if strings.Contains(URL, "amazon") {
		return amazon.New(), structs.WEBSHOP_AMAZON, nil
	}

	return nil, structs.WEBSHOP_NONE, fmt.Errorf("No webshop interface exists for this website")

}

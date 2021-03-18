package switcher

import (
	"dolos-dev/pkg/driver/webshop"
	"dolos-dev/pkg/driver/webshop/amazon"
	"dolos-dev/pkg/structs"
	"fmt"
	"strings"
)

//GetWebshop is a switcher function that gets the correct webshop driver using the given URL
func GetWebshop(URL string) (webshop.Webshop, structs.Webshop, error) {
	if strings.Contains(URL, "amazon.com") {
		return amazon.New(structs.WEBSHOP_AMAZON), structs.WEBSHOP_AMAZON, nil
	}

	if strings.Contains(URL, "amazon.nl") {
		return amazon.New(structs.WEBSHOP_AMAZONNL), structs.WEBSHOP_AMAZONNL, nil
	}

	if strings.Contains(URL, "amazon.de") {
		return amazon.New(structs.WEBSHOP_AMAZONDE), structs.WEBSHOP_AMAZONDE, nil
	}

	if strings.Contains(URL, "amazon.it") {
		return amazon.New(structs.WEBSHOP_AMAZONIT), structs.WEBSHOP_AMAZONIT, nil
	}

	if strings.Contains(URL, "amazon.fr") {
		return amazon.New(structs.WEBSHOP_AMAZONFR), structs.WEBSHOP_AMAZONFR, nil
	}

	return nil, structs.WEBSHOP_NONE, fmt.Errorf("No webshop interface exists for this website")

}

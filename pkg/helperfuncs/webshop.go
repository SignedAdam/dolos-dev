package helperfuncs

import (
	"strings"
	"dolos-dev/pkg/structs"

)

func GetWebshopFromString(url string) structs.Webshop{

	if strings.Contains(url, "amazon.com") {
		return structs.WEBSHOP_AMAZON
	}
	if strings.Contains(url, "amazon.fr") {
		return structs.WEBSHOP_AMAZONFR
	}
	if strings.Contains(url, "amazon.nl") {
		return structs.WEBSHOP_AMAZONNL
	}
	if strings.Contains(url, "amazon.de") {
		return structs.WEBSHOP_AMAZONDE
	}
	if strings.Contains(url, "amazon.it") {
		return structs.WEBSHOP_AMAZONIT
	}

	return structs.WEBSHOP_NONE
}
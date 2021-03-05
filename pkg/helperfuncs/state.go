package helperfuncs

import (
	"adam/learn-gitlab/pkg/structs"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

//SaveState saves the Product URLs config to a json file
func SaveState(products []*structs.ProductURL) error {
	//check if directory exists already
	_, err := os.Stat("stockalert-config/")
	if os.IsNotExist(err) { //directory doesn't exist -> create it
		err = os.Mkdir("stockalert-config/", os.ModeDir)
		if err != nil {
			return fmt.Errorf("Failed to create new directory (%v)", err)
		}
	} //TODO: add an error check if there is an error but it's not os.IsNotExist

	//marshal our URLs to json
	prettyJSON, err := json.MarshalIndent(products, "", " ")
	if err != nil {
		return fmt.Errorf("Failed to pretty marshal Product URLs config to json (%v)", err)
	}

	//write json to file
	err = ioutil.WriteFile("stockalert-config/config.json", prettyJSON, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write json to file (%v)", err)
	}

	return nil
}

//LoadState populates the Product URLs config from a json file
func LoadState() (productURLs []*structs.ProductURL, err error) {
	_, err = os.Stat("stockalert-config/config.json")
	if os.IsNotExist(err) {
		//directory doesn't exist -> return nil

		return nil, nil
	}

	//read json from file and unmarshal to our Product URLs config
	jsonURLs, _ := ioutil.ReadFile("stockalert-config/config.json")
	err = json.Unmarshal(jsonURLs, &productURLs)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal json to Product URLs config (%v)", err)
	}

	return productURLs, nil
}

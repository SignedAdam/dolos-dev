package helperfuncs

import (
	"bufio"
	"dolos-dev/pkg/structs"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func SaveBodyToHTML(bodyBytes []byte) error {
	readStr := string(bodyBytes)
	fmt.Println("writing to file:", readStr)
	rndStr := GenerateRandomString(10)
	err := ioutil.WriteFile(fmt.Sprintf("debug/%s.html", rndStr), []byte(readStr), 0644)
	if err != nil {
		return fmt.Errorf("failed to write body to HTML file (%v)", err)
	}

	return nil
}

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
	prettyJSON, err := json.MarshalIndent(products, "", "\t")
	if err != nil {
		return fmt.Errorf("Failed to pretty marshal Product URLs config to json (%v)", err)
	}

	//write json to file
	err = ioutil.WriteFile("stockalert-config/product-config.json", prettyJSON, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write json to file (%v)", err)
	}

	return nil
}

//LoadState populates the Product URLs config from a json file
func LoadState(productURLs *[]*structs.ProductURL) (err error) {
	_, err = os.Stat("stockalert-config/product-config.json")
	if os.IsNotExist(err) {
		//directory doesn't exist -> return nil

		return nil
	}

	//read json from file and unmarshal to our Product URLs config
	jsonURLs, _ := ioutil.ReadFile("stockalert-config/product-config.json")
	err = json.Unmarshal(jsonURLs, productURLs)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal json to Product URLs config (%v)", err)
	}

	return nil
}

//LoadGlobalConfig reads the global-config.json file and loads the necessary parameters
func LoadGlobalConfig(config **structs.GlobalConfig) (err error) {
	_, err = os.Stat("stockalert-config/global-config.json")
	if os.IsNotExist(err) {
		return fmt.Errorf("global-config.json does not exist")
	}

	//read json from file and unmarshal to global config struct
	jsonURLs, _ := ioutil.ReadFile("stockalert-config/global-config.json")
	err = json.Unmarshal(jsonURLs, &config)
	if err != nil {
		return fmt.Errorf("Failed to unmarshal json to global config (%v)", err)
	}

	return nil
}

//LoadProxyConfigs reads the proxy-config.json file and loads all the proxies into a list
func LoadProxyConfigs(proxies *[]*structs.Proxy) (err error) {
	_, err = os.Stat("stockalert-config/proxy-config.json")
	if !os.IsNotExist(err) { //file exists
		//read json from file and unmarshal to our proxies list
		jsonProxies, _ := ioutil.ReadFile("stockalert-config/proxy-config.json")
		err = json.Unmarshal(jsonProxies, proxies)
		if err != nil {
			return fmt.Errorf("Failed to unmarshal json to proxies list (%v)", err)
		}
	}

	//load new proxies
	_, err = os.Stat("stockalert-config/new_proxies.txt")
	if os.IsNotExist(err) {
		return nil
	}
	proxiesSaved := false

	//read new proxies file
	file, err := os.Open("stockalert-config/new_proxies.txt")
	if err != nil {
		return fmt.Errorf("Failed to open new_proxies.txt (%v)", err)
	}

	defer func() {
		if proxiesSaved {
			err := os.RemoveAll("stockalert-config/new_proxies.txt")
			if err != nil {
				fmt.Println(err)
			}
		}
	}()
	defer file.Close()

	derefProxies := *proxies
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		proxyLine := scanner.Text()
		proxyParams := strings.Split(proxyLine, ":")
		if len(proxyParams) < 4 {
			break
		}

		derefProxies = append(derefProxies, &structs.Proxy{
			IP:       proxyParams[0],
			Port:     proxyParams[1],
			User:     proxyParams[2],
			Password: proxyParams[3],
		})
	}

	err = saveProxies(derefProxies)
	if err != nil {
		return fmt.Errorf("Failed to save new proxies to proxies json file (%v)", err)
	}
	proxiesSaved = true

	*(proxies) = derefProxies
	return nil
}

func saveProxies(proxies []*structs.Proxy) error {
	//check if directory exists already
	_, err := os.Stat("stockalert-config/proxy-config.json")
	if os.IsNotExist(err) { //directory doesn't exist -> create it
		err = os.Mkdir("stockalert-config/proxy-config.json", os.ModeDir)
		if err != nil {
			return fmt.Errorf("Failed to create new directory (%v)", err)
		}
	}

	prettyJSON, err := json.MarshalIndent(proxies, "", "\t")
	if err != nil {
		return fmt.Errorf("Failed to pretty marshal proxies to json (%v)", err)
	}

	//write json to file
	err = ioutil.WriteFile("stockalert-config/proxy-config.json", prettyJSON, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write json to file (%v)", err)
	}

	return nil
}

//LoadAllConfigs loads all necessary config files - this is just a helper function to avoid having to call every single config loader separately
func LoadAllConfigs(productURLs *[]*structs.ProductURL, config **structs.GlobalConfig, proxies *[]*structs.Proxy) (err error) {
	err = LoadState(productURLs)
	if err != nil {
		return err
	}
	err = LoadGlobalConfig(config)
	if err != nil {
		return err
	}
	err = LoadProxyConfigs(proxies)
	if err != nil {
		return err
	}

	return nil
}

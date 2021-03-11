package helperfuncs

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

//GetBodyHTML retrieves the HTML body of the given URL.
//Will use proxy if proxy auth info is provided (at least ProxyIP and ProxyPort are required)
//Returns a body read closer or an error if any
func GetBodyHTML(rawURL string, proxyIP, proxyPort, proxyUser, proxyPassword string) (io.ReadCloser, error) {
	var httpClient *http.Client
	if proxyIP != "" {
		proxyStr := "http://"
		//http://user:password@prox-server:3128
		if proxyUser != "" {
			proxyStr = fmt.Sprintf("%s%s:%s@", proxyStr, proxyUser, proxyPassword)
		}

		proxyStr = fmt.Sprintf("%s%s:%s", proxyStr, proxyIP, proxyPort)

		proxyURL, err := url.Parse(proxyStr)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse proxy URL %s (%v)", proxyStr, err)
		}

		var PTransport = &http.Transport{
			//Proxy: ProxyFunc(FromStr(proxyStr)),
			Proxy: http.ProxyURL(proxyURL),
		}

		httpClient = &http.Client{
			Transport: PTransport,
		}
	} else {
		httpClient = http.DefaultClient
	}

	destURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse URL %s (%v)", rawURL, err)
	}

	//create HTTP request
	req := &http.Request{
		Method: "GET",
		URL:    destURL,
		Header: map[string][]string{
			"Content-Type": {"application/text; charset=UTF-8"},
			"User-Agent":   {"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:66.0) Gecko/20100101 Firefox/66.0"},
			//"Accept-Encoding":           {"gzip, deflate"},
			"DNT":                       {"1"},
			"Connection":                {"close"},
			"Upgrade-Insecure-Requests": {"1"},
		},
	}

	//execute http request
	resp, err := httpClient.Do(req)
	// handle the error if there is one
	if err != nil {
		return nil, fmt.Errorf("Failed to make GET request to %s (%v)", rawURL, err)
	}

	return resp.Body, nil
}

package helperfuncs

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func GetBodyHTML(rawURL string) (io.ReadCloser, error) {

	destURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse URL %s (%v)", rawURL, err)
	}

	fmt.Println(destURL.String())
	//create HTTP request
	req := &http.Request{
		Method: "GET",
		URL:    destURL,
		Header: map[string][]string{
			"Content-Type": {"application/text; charset=UTF-8"},
		},
	}

	//execute http request
	resp, err := http.DefaultClient.Do(req)
	//resp, err := http.Get(rawURL)
	// handle the error if there is one
	if err != nil {
		return nil, fmt.Errorf("Failed to make GET request to %s (%v)", rawURL, err)
	}

	return resp.Body, nil
	/*
		//go through the body and find all URLs
		tokenizer := html.NewTokenizer(resp.Body)

		return tokenizer, nil
	*/
}

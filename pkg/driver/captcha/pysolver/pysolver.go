package pysolver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type response struct {
	output string `json:"output"`
}

//SolveCaptcha takes a captcha image URL and a endpoint where captcha images are solved. Returns solved captcha and error if any
func SolveCaptcha(captchaURL, solverEndpoint string) (string, error) {
	req, err := http.NewRequest("POST", solverEndpoint, nil)
	req.Header.Set("captchaURL", captchaURL)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failed to connect to solver endpoint (%v)", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	response := response{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal captcha solver response (%v)", err)
	}

	return response.output, nil
}

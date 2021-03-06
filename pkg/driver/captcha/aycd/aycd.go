package aycd

import (
	"fmt"

	"gitlab.com/aycd-inc/autosolve-clients/autosolve-client-go"
)

type CaptchaSolver struct {
}

func New() (*CaptchaSolver, error) {
	accessToken := ""
	apiKey := ""
	//First, create listener functions, which will be passed to the module and called when tokens are received/cancelled

	//Establishes listener function to receive token responses
	var tokenListener autosolve.CaptchaTokenResponseListener = func(tokenResponse autosolve.CaptchaTokenResponse) {
		//handle token response here
	}

	//Establishes listener function to receive cancel token responses
	var tokenCancelListener autosolve.CaptchaTokenCancelResponseListener = func(cancelTokenResponse autosolve.CaptchaTokenCancelResponse) {
		//handle cancel response here
	}

	//Establishes listener function to receive status updates from AutoSolve
	var statusListener autosolve.StatusListener = func(status autosolve.Status) {
		//handle status message here
	}

	//Establishes listener function to receive errors
	var errorListener autosolve.ErrorListener = func(err error) {
		//handle error message here
	}

	//To initiate the Go module, call the following function, with the functions we defined above:
	err := autosolve.Load("your-client-id", tokenListener, tokenCancelListener, statusListener, errorListener)
	if err != nil {
		return nil, fmt.Errorf("Failed to init the captcha solver (%v)", err)
	}

	//Then, to initiate the connection (and to update user credentials):
	result, err := autosolve.Connect(accessToken, apiKey)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to captcha solver (%v)", err)
	}

	//Result is a string "enum", outlined as a type ConnectResult with multiple types. You can process the response below

	switch result {
	case autosolve.Success:
		print("Successful Connection")
	case autosolve.InvalidClientId:
		return nil, fmt.Errorf("Invalid Client Key")
	case autosolve.InvalidAccessToken:
		return nil, fmt.Errorf("Invalid access token")
	case autosolve.InvalidApiKey:
		return nil, fmt.Errorf("Invalid Api Key")
	case autosolve.InvalidCredentials:
		return nil, fmt.Errorf("Invalid Credentials")
	}

	solver := &CaptchaSolver{}

	return solver, nil
}
func solveCaptcha(url string) {
	autosolve.SendTokenRequest(autosolve.CaptchaTokenRequest{})
}

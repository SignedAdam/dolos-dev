package aycd

import (
	"adam/learn-gitlab/pkg/helperfuncs"
	"fmt"

	"gitlab.com/aycd-inc/autosolve-clients/autosolve-client-go"
)

//CaptchaSolver represents an instance of this driver
type CaptchaSolver struct {
	solvedTokens map[string]chan string
}

//New creates a new instance of the AYCD captcha solver driver
func New() (*CaptchaSolver, error) {
	solver := &CaptchaSolver{
		solvedTokens: make(map[string]chan string),
	}
	accessToken := "192703-457cc672-27ad-49c5-b415-fe3f111768af"
	apiKey := "05e99354-c88c-4529-9cda-7b18d42e091d"

	//init listener callbacks
	var tokenListener autosolve.CaptchaTokenResponseListener = solver.handleResponse
	var tokenCancelListener autosolve.CaptchaTokenCancelResponseListener = solver.handleCancel
	var statusListener autosolve.StatusListener = solver.handleStatusUpdate
	var errorListener autosolve.ErrorListener = solver.handleError

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

	switch result {
	case autosolve.Success:
		fmt.Println("Successful Connection")
	case autosolve.InvalidClientId:
		return nil, fmt.Errorf("Invalid Client Key")
	case autosolve.InvalidAccessToken:
		return nil, fmt.Errorf("Invalid access token")
	case autosolve.InvalidApiKey:
		return nil, fmt.Errorf("Invalid Api Key")
	case autosolve.InvalidCredentials:
		return nil, fmt.Errorf("Invalid Credentials")
	}

	return solver, nil
}
func (solver *CaptchaSolver) solveCaptcha(url string, responseToken chan string) error {

	request := autosolve.CaptchaTokenRequest{
		TaskId: helperfuncs.GenerateRandomString(10),

		Url: url,

		//Public ReCaptcha key for a given site
		SiteKey: "TODO TODO TODO",

		//Version of Captcha
		//Options:
		/*
		  reCaptcha V2 Checkbox is 0
		  reCaptcha V2 Invisible is 1
		  reCaptcha V3 Score is 2
		  hCaptcha Checkbox is 3
		  hCaptcha Invisible is 4
		  GeeTest is 5
		*/
		Version: 0,
	}
	err := autosolve.SendTokenRequest(request)
	if err != nil {
		return fmt.Errorf("Failed to send request to solve captcha (%v)", err)
	}

	solver.solvedTokens[request.TaskId] = responseToken

	return nil

}

func (solver *CaptchaSolver) handleResponse(tokenResponse autosolve.CaptchaTokenResponse) {
	solver.solvedTokens[tokenResponse.TaskId] <- tokenResponse.Token
}

func (solver *CaptchaSolver) handleCancel(cancelTokenResponse autosolve.CaptchaTokenCancelResponse) {
	for _, request := range cancelTokenResponse.Requests {
		solver.solvedTokens[request.TaskId] <- "1"
	}
}

func (solver *CaptchaSolver) handleStatusUpdate(status autosolve.Status) {
	fmt.Println(fmt.Errorf("AYCD: connection status updated: %v", status))
}

func (solver *CaptchaSolver) handleError(err error) {
	fmt.Println(fmt.Errorf("AYCD: error (%v)", err))
}

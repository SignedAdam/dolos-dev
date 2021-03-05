package helperfuncs

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/0x434D53/openinbrowser"
)

//OpenInBrowser opens a given filepath in the default browser
func OpenInBrowser(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("HTML file not found :( (%s)", path)
	}
	openinbrowser.Open(path)
	return nil
}

//CreateSessionHTML creates a new html file for the given session based on the captchasolver.html template
func CreateSessionHTML(sessionID, captchaURL string) error {
	//read env var to get our template file's path
	templatePath := os.Getenv("CAPTCHA_HTML_TEMPLATE_PATH")
	if templatePath == "" {
		return fmt.Errorf("CAPTCHA_HTML_TEMPLATE_PATH env var missing")
	}

	//read the template file
	readBytes, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template HTML file (%v)", err)
	}

	readStr := string(readBytes)

	//replace the captcha URL template string with relevant data
	readStr = strings.Replace(readStr, "{captchaURL}", captchaURL, 1)

	//replace the session ID template string with relevant data
	readStr = strings.Replace(readStr, "{captchaSessionID}", sessionID, 1)

	//write the result into a new file under captchatemplates/[sessionid].html
	err = ioutil.WriteFile(fmt.Sprintf("captchatemplates/%s.html", sessionID), []byte(readStr), 0644)
	if err != nil {
		return fmt.Errorf("failed to create a captcha solver session HTML file (%v)", err)
	}

	return nil
}

package selenium

import (
	"archive/zip"
	"dolos-dev/pkg/helperfuncs"
	"dolos-dev/pkg/structs"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func createPluginZip(proxy structs.Proxy) (string, error) {
	sessionID := helperfuncs.GenerateRandomString(7)
	newZipFile, err := os.Create(fmt.Sprint("selenium/plugin", sessionID, ".zip"))
	if err != nil {
		return "", err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	err = os.Mkdir(fmt.Sprintf("selenium/%s/", sessionID), os.ModeDir)
	if err != nil {
		return "", fmt.Errorf("Failed to create new directory (%v)", err)
	}

	//read the template file
	readBytes, err := ioutil.ReadFile("selenium/proxy/background_template.js")
	if err != nil {
		return "", fmt.Errorf("failed to read template HTML file (%v)", err)
	}

	readStr := string(readBytes)

	readStr = strings.Replace(readStr, "{PROXYIP}", proxy.IP, 1)
	readStr = strings.Replace(readStr, "{PROXYPORT}", proxy.Port, 1)
	readStr = strings.Replace(readStr, "{PROXYUSER}", proxy.User, 1)
	readStr = strings.Replace(readStr, "{PROXYPASS}", proxy.Password, 1)

	//write the result into a new file under captchatemplates/[sessionid].html
	err = ioutil.WriteFile(fmt.Sprintf("selenium/%s/background.js", sessionID), []byte(readStr), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create a background.js file (%v)", err)
	}

	if err = AddFileToZip(zipWriter, fmt.Sprintf("selenium/%s/background.js", sessionID), "background.js"); err != nil {
		return "", err
	}

	if err = AddFileToZip(zipWriter, "selenium/proxy/manifest.json", "manifest.json"); err != nil {
		return "", err
	}

	return fmt.Sprintf("selenium/plugin%s.zip", sessionID), nil
}

func AddFileToZip(zipWriter *zip.Writer, filePath, fileName string) error {

	fileToZip, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	// Get the file information
	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = fileName
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}

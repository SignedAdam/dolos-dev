package helperfuncs

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
)

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

func DeleteFileOrDir(path string) error {
	err := os.RemoveAll(path)
	return err
}

func SaveImage(metadata string, imageBytes []byte) (string, error) {
	// Encode to `PNG` with `DefaultCompression` level
	// then save to file

	img, _, err := image.Decode(bytes.NewReader(imageBytes))

	imagePath := fmt.Sprintf("screenshots/screenshot_%s_%s.png", metadata, GenerateRandomString(5))
	f, err := os.Create(imagePath)
	err = png.Encode(f, img)
	if err != nil {
		return "", fmt.Errorf("Failed to save image (%v)", err)
	}

	return imagePath, nil
}

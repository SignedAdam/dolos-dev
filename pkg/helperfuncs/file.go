package helperfuncs

import (
	"archive/zip"
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

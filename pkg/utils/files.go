package utils

import (
	"crypto/rand"
	"encoding/hex"
	"io/ioutil"
)

// CreateTempFile creates random tmeporary file and stores the content to the file and return path to it or error
func CreateTempFile(content []byte) (string, error) {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	tmpfile, err := ioutil.TempFile("", hex.EncodeToString(randBytes))
	if err != nil {
		return "", err
	}
	defer tmpfile.Close()

	if _, err := tmpfile.Write(content); err != nil {
		return "", err
	}

	return tmpfile.Name(), nil
}

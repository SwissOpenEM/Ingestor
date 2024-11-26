package webserver

import (
	"crypto/rand"
	"encoding/base64"
)

func generateRandomByteSlice(len uint) ([]byte, error) {
	b := make([]byte, len)
	_, err := rand.Read(b)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

func generateRandomString(len uint) (string, error) {
	b, err := generateRandomByteSlice(len)
	return base64.URLEncoding.EncodeToString(b), err
}

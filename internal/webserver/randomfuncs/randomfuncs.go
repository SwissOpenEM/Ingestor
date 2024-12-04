package randomfuncs

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateRandomByteSlice(len uint) ([]byte, error) {
	b := make([]byte, len)
	_, err := rand.Read(b)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

func GenerateRandomString(len uint) (string, error) {
	b, err := GenerateRandomByteSlice(len)
	return base64.URLEncoding.EncodeToString(b), err
}

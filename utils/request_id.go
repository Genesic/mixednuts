package utils

import (
	"crypto/rand"
	"encoding/hex"
)

func GenRequestID() string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(bytes)
}

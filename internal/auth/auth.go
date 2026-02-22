package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

const KeyLength = 64
const KeyPrefix = "artf_"

// GenerateKey generates a random key
func GenerateKey() string {
	b := make([]byte, KeyLength)
	rand.Read(b)
	return KeyPrefix + base64.URLEncoding.EncodeToString(b)
}

func HashKey(key string) []byte {
	hash := sha256.Sum256([]byte(strings.TrimPrefix(key, KeyPrefix)))
	return hash[:]
}

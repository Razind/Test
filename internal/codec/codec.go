package codec

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

const codeLength = 8

func NewCode() (string, error) {
	raw := make([]byte, codeLength)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	return strings.TrimRight(encoded, "=")[:codeLength], nil
}

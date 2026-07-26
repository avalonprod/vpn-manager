package peers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const accessTokenBytes = 32

func NewAccessToken() (string, error) {
	buf := make([]byte, accessTokenBytes)

	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate access token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

package operations

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const DefaultCapabilityTokenBytes = 32

func GenerateCapabilityToken() (string, error) {
	return GenerateCapabilityTokenWithBytes(DefaultCapabilityTokenBytes)
}

func GenerateCapabilityTokenWithBytes(size int) (string, error) {
	if size < DefaultCapabilityTokenBytes {
		return "", fmt.Errorf("capability token size must be at least %d bytes", DefaultCapabilityTokenBytes)
	}

	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

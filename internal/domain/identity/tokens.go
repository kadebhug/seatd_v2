package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"strings"
)

const (
	LookupPrefixLength      = 16
	PairingCodeLookupLength = 4
	PairingCodeRandomBytes  = 5
	SecretBytes             = 32
)

var ErrInvalidSecret = errors.New("invalid secret")

func GenerateOpaqueSecret() (string, error) {
	buf := make([]byte, SecretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func GeneratePairingCode() (string, error) {
	buf := make([]byte, PairingCodeRandomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return strings.ToUpper(base64.RawStdEncoding.EncodeToString(buf)), nil
}

func LookupPrefix(secret string) (string, error) {
	secret = strings.TrimSpace(secret)
	if len(secret) < LookupPrefixLength {
		return "", ErrInvalidSecret
	}
	return secret[:LookupPrefixLength], nil
}

func PairingCodeLookupPrefix(code string) (string, error) {
	code = NormalizePairingCode(code)
	if len(code) < PairingCodeLookupLength {
		return "", ErrInvalidSecret
	}
	return code[:PairingCodeLookupLength], nil
}

func NormalizePairingCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(code), "-", ""))
}

func HashSecret(secret string) ([]byte, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, ErrInvalidSecret
	}
	sum := sha256.Sum256([]byte(secret))
	return sum[:], nil
}

func SecretMatches(secret string, expectedHash []byte) (bool, error) {
	actualHash, err := HashSecret(secret)
	if err != nil {
		return false, err
	}
	if len(actualHash) != len(expectedHash) {
		return false, nil
	}
	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1, nil
}

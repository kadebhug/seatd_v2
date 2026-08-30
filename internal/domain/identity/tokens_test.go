package identity

import (
	"errors"
	"testing"
)

func TestSecretHashAndMatch(t *testing.T) {
	secret, err := GenerateOpaqueSecret()
	if err != nil {
		t.Fatalf("generate opaque secret: %v", err)
	}
	if len(secret) < LookupPrefixLength {
		t.Fatalf("secret length = %d, want at least %d", len(secret), LookupPrefixLength)
	}

	prefix, err := LookupPrefix(secret)
	if err != nil {
		t.Fatalf("lookup prefix: %v", err)
	}
	if len(prefix) != LookupPrefixLength {
		t.Fatalf("prefix length = %d, want %d", len(prefix), LookupPrefixLength)
	}

	hash, err := HashSecret(secret)
	if err != nil {
		t.Fatalf("hash secret: %v", err)
	}
	matches, err := SecretMatches(secret, hash)
	if err != nil {
		t.Fatalf("secret matches: %v", err)
	}
	if !matches {
		t.Fatalf("secret did not match its hash")
	}

	matches, err = SecretMatches(secret+"x", hash)
	if err != nil {
		t.Fatalf("secret mismatch check: %v", err)
	}
	if matches {
		t.Fatalf("modified secret matched hash")
	}
}

func TestLookupPrefixRejectsShortSecret(t *testing.T) {
	_, err := LookupPrefix("short")
	if !errors.Is(err, ErrInvalidSecret) {
		t.Fatalf("lookup prefix error = %v, want invalid secret", err)
	}
}

func TestPairingCodeLookupPrefixNormalizesInput(t *testing.T) {
	code, err := GeneratePairingCode()
	if err != nil {
		t.Fatalf("generate pairing code: %v", err)
	}
	if len(code) < PairingCodeLookupLength {
		t.Fatalf("pairing code length = %d, want at least %d", len(code), PairingCodeLookupLength)
	}

	prefix, err := PairingCodeLookupPrefix(" " + code[:4] + "-" + code[4:] + " ")
	if err != nil {
		t.Fatalf("pairing lookup prefix: %v", err)
	}
	if prefix != code[:PairingCodeLookupLength] {
		t.Fatalf("prefix = %q, want %q", prefix, code[:PairingCodeLookupLength])
	}
}

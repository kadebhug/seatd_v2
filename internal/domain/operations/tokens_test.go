package operations

import "testing"

func TestGenerateCapabilityTokenWithBytes(t *testing.T) {
	token, err := GenerateCapabilityTokenWithBytes(DefaultCapabilityTokenBytes)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	if token == "" {
		t.Fatalf("token is empty")
	}

	other, err := GenerateCapabilityTokenWithBytes(DefaultCapabilityTokenBytes)
	if err != nil {
		t.Fatalf("generate second token: %v", err)
	}
	if token == other {
		t.Fatalf("generated duplicate tokens")
	}
}

func TestGenerateCapabilityTokenWithBytesRejectsSmallSize(t *testing.T) {
	_, err := GenerateCapabilityTokenWithBytes(DefaultCapabilityTokenBytes - 1)
	if err == nil {
		t.Fatalf("small token size succeeded")
	}
}

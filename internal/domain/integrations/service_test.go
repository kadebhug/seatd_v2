package integrations

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
)

func TestReferencePOSAdapterHandleWebhook(t *testing.T) {
	payload := []byte(`{"eventId":"evt-1","externalTableId":"ref-t1","status":"occupied","partySize":4}`)
	secret := "test-reference-secret"
	signature := hmacSignature(secret, payload)

	tests := []struct {
		name          string
		signature     string
		wantValid     bool
		wantStatus    string
		wantPartySize int32
	}{
		{
			name:          "valid signature returns canonical fact",
			signature:     signature,
			wantValid:     true,
			wantStatus:    "occupied",
			wantPartySize: 4,
		},
		{
			name:          "invalid signature keeps canonical fact but marks invalid",
			signature:     "bad-signature",
			wantValid:     false,
			wantStatus:    "occupied",
			wantPartySize: 4,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SEATD_REFERENCE_POS_WEBHOOK_SECRET", secret)

			fact, valid, err := ReferencePOSAdapter{}.HandleWebhook(context.Background(), Connection{}, payload, tt.signature)
			if err != nil {
				t.Fatalf("HandleWebhook() error = %v", err)
			}
			if valid != tt.wantValid {
				t.Fatalf("signature valid = %v, want %v", valid, tt.wantValid)
			}
			if fact.ExternalEventID != "evt-1" || fact.ExternalTableID != "ref-t1" || fact.Status != tt.wantStatus {
				t.Fatalf("fact = %#v", fact)
			}
			if fact.PartySize == nil || *fact.PartySize != tt.wantPartySize {
				t.Fatalf("party size = %v, want %d", fact.PartySize, tt.wantPartySize)
			}
		})
	}
}

func TestReferencePOSAdapterRejectsInvalidWebhook(t *testing.T) {
	_, _, err := ReferencePOSAdapter{}.HandleWebhook(
		context.Background(),
		Connection{},
		[]byte(`{"eventId":"evt-1","externalTableId":"ref-t1","status":"reserved"}`),
		"",
	)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("HandleWebhook() error = %v, want %v", err, ErrValidation)
	}
}

func TestReferencePOSFetchExternalStatus(t *testing.T) {
	connection := Connection{
		Config: []byte(`{"referenceTables":[{"externalTableId":"ref-t1","label":"T1","status":"available"},{"externalTableId":"ref-bad","status":"reserved"}]}`),
	}
	states, err := ReferencePOSAdapter{}.FetchExternalStatus(context.Background(), connection)
	if err != nil {
		t.Fatalf("FetchExternalStatus() error = %v", err)
	}
	if len(states) != 1 {
		t.Fatalf("states len = %d, want 1", len(states))
	}
	if states[0].ExternalTableID != "ref-t1" || states[0].Status != "available" || states[0].Label != "T1" {
		t.Fatalf("state = %#v", states[0])
	}
}

func hmacSignature(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

package realtime

import (
	"net/http"
	"testing"

	"github.com/kadebhug/seatd_v2/internal/domain/identity"
)

func TestBearerCredential(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{name: "missing", want: ""},
		{name: "bearer credential", header: "Bearer secret-value", want: "secret-value"},
		{name: "case insensitive scheme", header: "bearer secret-value", want: "secret-value"},
		{name: "wrong scheme", header: "Basic secret-value", want: ""},
		{name: "malformed", header: "secret-value", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/v1/realtime", nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			if got := bearerCredential(req); got != tt.want {
				t.Fatalf("bearerCredential() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeviceAllowsOperationsRead(t *testing.T) {
	tests := []struct {
		name       string
		deviceType string
		want       bool
	}{
		{name: "waiter mobile", deviceType: identity.DeviceTypeWaiterMobile, want: true},
		{name: "manager tablet", deviceType: identity.DeviceTypeManagerTablet, want: true},
		{name: "display", deviceType: identity.DeviceTypeDisplay, want: true},
		{name: "host device", deviceType: identity.DeviceTypeHostDevice, want: true},
		{name: "unknown", deviceType: "unknown", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deviceAllowsOperationsRead(tt.deviceType); got != tt.want {
				t.Fatalf("deviceAllowsOperationsRead(%q) = %v, want %v", tt.deviceType, got, tt.want)
			}
		})
	}
}

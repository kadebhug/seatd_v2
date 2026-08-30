package app

import (
	"log/slog"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Setenv("SEATD_ENV", "test")
	t.Setenv("SEATD_LOG_LEVEL", "debug")
	t.Setenv("SEATD_API_PORT", "19080")
	t.Setenv("SEATD_SHUTDOWN_TIMEOUT", "2s")

	cfg, err := LoadConfig("api", 8080, BuildInfo{Version: "v1.2.3"})
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Environment != EnvTest {
		t.Fatalf("Environment = %q, want %q", cfg.Environment, EnvTest)
	}
	if cfg.Address != ":19080" {
		t.Fatalf("Address = %q, want :19080", cfg.Address)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("LogLevel = %v, want %v", cfg.LogLevel, slog.LevelDebug)
	}
}

func TestLoadConfigRejectsInvalidEnvironment(t *testing.T) {
	t.Setenv("SEATD_ENV", "qa")

	_, err := LoadConfig("api", 8080, BuildInfo{})
	if err == nil {
		t.Fatal("LoadConfig() error = nil, want error")
	}
}

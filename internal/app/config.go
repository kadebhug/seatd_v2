package app

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvLocal      = "local"
	EnvTest       = "test"
	EnvStaging    = "staging"
	EnvProduction = "production"
)

type Config struct {
	Name                   string
	Environment            string
	Address                string
	DatabaseURL            string
	GuestWebOrigin         string
	GuestAPIBaseURL        string
	InternalAPISecret      string
	TrustedIdentityHeaders bool
	LogLevel               slog.Level
	ShutdownTimeout        time.Duration
	Version                BuildInfo
}

type BuildInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func LoadConfig(serviceName string, defaultPort int, build BuildInfo) (Config, error) {
	if strings.TrimSpace(serviceName) == "" {
		return Config{}, errors.New("service name is required")
	}

	env := getEnv("SEATD_ENV", EnvLocal)
	if !validEnvironment(env) {
		return Config{}, fmt.Errorf("SEATD_ENV must be local, test, staging, or production: %q", env)
	}

	level, err := parseLogLevel(getEnv("SEATD_LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, err
	}

	port, err := getPort(serviceName, defaultPort)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := getDuration("SEATD_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	databaseURL := strings.TrimSpace(os.Getenv("SEATD_DATABASE_URL"))
	guestWebOrigin := strings.TrimSpace(os.Getenv("SEATD_GUEST_WEB_ORIGIN"))
	guestAPIBaseURL := strings.TrimSpace(os.Getenv("SEATD_GUEST_PUBLIC_API_BASE_URL"))
	internalAPISecret := strings.TrimSpace(os.Getenv("SEATD_INTERNAL_API_SECRET"))
	trustedIdentityHeaders, err := getBool("SEATD_TRUSTED_IDENTITY_HEADERS", false)
	if err != nil {
		return Config{}, err
	}

	build = normalizeBuildInfo(build)

	return Config{
		Name:                   serviceName,
		Environment:            env,
		Address:                fmt.Sprintf(":%d", port),
		DatabaseURL:            databaseURL,
		GuestWebOrigin:         guestWebOrigin,
		GuestAPIBaseURL:        guestAPIBaseURL,
		InternalAPISecret:      internalAPISecret,
		TrustedIdentityHeaders: trustedIdentityHeaders,
		LogLevel:               level,
		ShutdownTimeout:        shutdownTimeout,
		Version:                build,
	}, nil
}

func NewLogger(cfg Config) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})).With(
		"service", cfg.Name,
		"environment", cfg.Environment,
		"version", cfg.Version.Version,
		"commit", cfg.Version.Commit,
	)
}

func ServiceEnvKey(serviceName, suffix string) string {
	name := strings.NewReplacer("-", "_", " ", "_").Replace(strings.ToUpper(serviceName))
	return "SEATD_" + name + "_" + suffix
}

func getPort(serviceName string, fallback int) (int, error) {
	key := ServiceEnvKey(serviceName, "PORT")
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		value = strings.TrimSpace(os.Getenv("SEATD_PORT"))
	}
	if value == "" {
		return fallback, nil
	}

	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("%s must be a valid TCP port: %q", key, value)
	}
	return port, nil
}

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration such as 10s: %w", key, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}
	return duration, nil
}

func getBool(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	switch strings.ToLower(value) {
	case "1", "true", "t", "yes", "y", "on":
		return true, nil
	case "0", "false", "f", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be a boolean: %q", key, value)
	}
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func validEnvironment(env string) bool {
	switch env {
	case EnvLocal, EnvTest, EnvStaging, EnvProduction:
		return true
	default:
		return false
	}
}

func parseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("SEATD_LOG_LEVEL must be debug, info, warn, or error: %q", value)
	}
}

func normalizeBuildInfo(build BuildInfo) BuildInfo {
	if build.Version == "" {
		build.Version = "dev"
	}
	if build.Commit == "" {
		build.Commit = "unknown"
	}
	if build.Date == "" {
		build.Date = "unknown"
	}
	return build
}

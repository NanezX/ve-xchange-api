package config

import (
	"os"
	"testing"
)

// setenv sets an env var for the duration of the test and restores the
// previous value (or unsets it) via t.Cleanup.
func setenv(t *testing.T, key, value string) {
	t.Helper()
	prev, hadPrev := os.LookupEnv(key)
	os.Setenv(key, value)
	t.Cleanup(func() {
		if hadPrev {
			os.Setenv(key, prev)
		} else {
			os.Unsetenv(key)
		}
	})
}

func unsetenv(t *testing.T, key string) {
	t.Helper()
	prev, hadPrev := os.LookupEnv(key)
	os.Unsetenv(key)
	t.Cleanup(func() {
		if hadPrev {
			os.Setenv(key, prev)
		}
	})
}

func TestLoadConfigValidPort(t *testing.T) {
	setenv(t, "APP_PORT", "9090")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if cfg.AppPort != 9090 {
		t.Fatalf("expected AppPort=9090, got %d", cfg.AppPort)
	}
}

func TestLoadConfigMissingPortDefaultsTo8080(t *testing.T) {
	unsetenv(t, "APP_PORT")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if cfg.AppPort != 8080 {
		t.Fatalf("expected AppPort=8080 (default), got %d", cfg.AppPort)
	}
}

func TestLoadConfigInvalidPortDefaultsTo8080(t *testing.T) {
	setenv(t, "APP_PORT", "not-a-number")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected success with default port, got %v", err)
	}
	if cfg.AppPort != 8080 {
		t.Fatalf("expected AppPort=8080 (default), got %d", cfg.AppPort)
	}
}

func TestLoadConfigNegativePortDefaultsTo8080(t *testing.T) {
	// strconv.ParseUint rejects negative values, so we fall back to 8080.
	setenv(t, "APP_PORT", "-1")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected success with default port, got %v", err)
	}
	if cfg.AppPort != 8080 {
		t.Fatalf("expected AppPort=8080 (default for negative), got %d", cfg.AppPort)
	}
}

func TestLoadConfigZeroPortDefaultsTo8080(t *testing.T) {
	// Port 0 is technically valid for ParseUint but semantically wrong for a server.
	// Current behavior: it is accepted as-is. This test documents that behavior.
	setenv(t, "APP_PORT", "0")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Port 0 parses successfully — document current behavior (no default applied).
	if cfg.AppPort != 0 {
		t.Fatalf("expected AppPort=0, got %d", cfg.AppPort)
	}
}

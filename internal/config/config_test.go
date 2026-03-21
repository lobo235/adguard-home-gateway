package config_test

import (
	"log/slog"
	"testing"

	"github.com/lobo235/adguard-home-gateway/internal/config"
)

// setRequired sets all required env vars; individual tests may blank one out.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("ADGUARD_SERVERS", "192.168.1.1")
	t.Setenv("ADGUARD_SCHEME", "")
	t.Setenv("ADGUARD_USER", "admin")
	t.Setenv("ADGUARD_PASSWORD", "secret")
	t.Setenv("ADGUARD_TLS_SKIP_VERIFY", "")
	t.Setenv("GATEWAY_API_KEY", "key123")
	t.Setenv("PORT", "")
	t.Setenv("LOG_LEVEL", "")
}

func TestLoad_Defaults(t *testing.T) {
	setRequired(t)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AdGuardScheme != "http" {
		t.Errorf("AdGuardScheme = %q, want http", cfg.AdGuardScheme)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.AdGuardTLSSkipVerify {
		t.Error("AdGuardTLSSkipVerify should default to false")
	}
}

func TestLoad_MultipleServers(t *testing.T) {
	setRequired(t)
	t.Setenv("ADGUARD_SERVERS", "192.168.1.1, 192.168.1.2:3000 , adguard.example.com")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"192.168.1.1", "192.168.1.2:3000", "adguard.example.com"}
	if len(cfg.AdGuardServers) != len(want) {
		t.Fatalf("servers = %v, want %v", cfg.AdGuardServers, want)
	}
	for i, s := range cfg.AdGuardServers {
		if s != want[i] {
			t.Errorf("servers[%d] = %q, want %q", i, s, want[i])
		}
	}
}

func TestLoad_OptionalAuth(t *testing.T) {
	setRequired(t)
	t.Setenv("ADGUARD_USER", "")
	t.Setenv("ADGUARD_PASSWORD", "")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("ADGUARD_USER and ADGUARD_PASSWORD should be optional, got: %v", err)
	}
	if cfg.AdGuardUser != "" || cfg.AdGuardPassword != "" {
		t.Error("expected empty credentials")
	}
}

func TestLoad_SchemeHTTPS(t *testing.T) {
	setRequired(t)
	t.Setenv("ADGUARD_SCHEME", "https")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AdGuardScheme != "https" {
		t.Errorf("AdGuardScheme = %q, want https", cfg.AdGuardScheme)
	}
}

func TestLoad_InvalidScheme(t *testing.T) {
	setRequired(t)
	t.Setenv("ADGUARD_SCHEME", "ftp")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for invalid ADGUARD_SCHEME")
	}
}

func TestLoad_TLSSkipVerify(t *testing.T) {
	setRequired(t)
	t.Setenv("ADGUARD_TLS_SKIP_VERIFY", "true")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.AdGuardTLSSkipVerify {
		t.Error("expected AdGuardTLSSkipVerify=true")
	}
}

func TestLoad_InvalidTLSSkipVerify(t *testing.T) {
	setRequired(t)
	t.Setenv("ADGUARD_TLS_SKIP_VERIFY", "notabool")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for invalid ADGUARD_TLS_SKIP_VERIFY")
	}
}

func TestLoad_LogLevels(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "warning", "error"} {
		t.Run(level, func(t *testing.T) {
			setRequired(t)
			t.Setenv("LOG_LEVEL", level)
			if _, err := config.Load(); err != nil {
				t.Errorf("LOG_LEVEL=%q should be valid, got: %v", level, err)
			}
		})
	}
}

func TestLoad_InvalidLogLevel(t *testing.T) {
	setRequired(t)
	t.Setenv("LOG_LEVEL", "verbose")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for invalid LOG_LEVEL")
	}
}

func TestLoad_PortOverride(t *testing.T) {
	setRequired(t)
	t.Setenv("PORT", "9090")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
}

func TestLoad_MissingServers(t *testing.T) {
	setRequired(t)
	t.Setenv("ADGUARD_SERVERS", "")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for missing ADGUARD_SERVERS")
	}
}

func TestLoad_MissingGatewayAPIKey(t *testing.T) {
	setRequired(t)
	t.Setenv("GATEWAY_API_KEY", "")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error for missing GATEWAY_API_KEY")
	}
}

func TestSlogLevel(t *testing.T) {
	cases := []struct {
		level string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"INFO", slog.LevelInfo}, // case-insensitive
	}
	for _, tc := range cases {
		setRequired(t)
		t.Setenv("LOG_LEVEL", tc.level)
		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("LOG_LEVEL=%q: unexpected error: %v", tc.level, err)
		}
		if got := cfg.SlogLevel(); got != tc.want {
			t.Errorf("LOG_LEVEL=%q: SlogLevel() = %v, want %v", tc.level, got, tc.want)
		}
	}
}

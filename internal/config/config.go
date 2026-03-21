package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// AdGuard Home upstream servers — one Client is created per entry.
	AdGuardServers      []string
	AdGuardScheme       string
	AdGuardUser         string
	AdGuardPassword     string
	AdGuardTLSSkipVerify bool

	// Gateway settings
	GatewayAPIKey string
	Port          string
	LogLevel      string
}

// SlogLevel converts the LogLevel string to a slog.Level.
func (c *Config) SlogLevel() slog.Level {
	switch strings.ToLower(c.LogLevel) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func Load() (*Config, error) {
	// Load .env if present — ignore error if file doesn't exist
	_ = godotenv.Load()

	cfg := &Config{
		AdGuardScheme:   os.Getenv("ADGUARD_SCHEME"),
		AdGuardUser:     os.Getenv("ADGUARD_USER"),
		AdGuardPassword: os.Getenv("ADGUARD_PASSWORD"),
		GatewayAPIKey:   os.Getenv("GATEWAY_API_KEY"),
		Port:            os.Getenv("PORT"),
		LogLevel:        os.Getenv("LOG_LEVEL"),
	}

	// ADGUARD_SERVERS — required, comma-separated host or host:port entries
	serversRaw := os.Getenv("ADGUARD_SERVERS")
	if serversRaw == "" {
		return nil, fmt.Errorf("ADGUARD_SERVERS is required")
	}
	for _, s := range strings.Split(serversRaw, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			cfg.AdGuardServers = append(cfg.AdGuardServers, s)
		}
	}
	if len(cfg.AdGuardServers) == 0 {
		return nil, fmt.Errorf("ADGUARD_SERVERS must contain at least one server address")
	}

	// ADGUARD_SCHEME — default http, must be http or https
	if cfg.AdGuardScheme == "" {
		cfg.AdGuardScheme = "http"
	}
	if cfg.AdGuardScheme != "http" && cfg.AdGuardScheme != "https" {
		return nil, fmt.Errorf("ADGUARD_SCHEME must be http or https, got %q", cfg.AdGuardScheme)
	}

	// ADGUARD_TLS_SKIP_VERIFY — default false
	if raw := os.Getenv("ADGUARD_TLS_SKIP_VERIFY"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("ADGUARD_TLS_SKIP_VERIFY must be true or false, got %q", raw)
		}
		cfg.AdGuardTLSSkipVerify = v
	}

	// GATEWAY_API_KEY — required
	if cfg.GatewayAPIKey == "" {
		return nil, fmt.Errorf("GATEWAY_API_KEY is required")
	}

	// PORT — default 8080
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	// LOG_LEVEL — default info, validate
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	switch strings.ToLower(cfg.LogLevel) {
	case "debug", "info", "warn", "warning", "error":
		// valid
	default:
		return nil, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error, got %q", cfg.LogLevel)
	}

	return cfg, nil
}

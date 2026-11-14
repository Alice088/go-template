package config

import (
	"strings"
	"testing"
	"time"
)

// validJWTSecret is a 32+ character secret accepted by validation.
const validJWTSecret = "0123456789abcdef0123456789abcdef"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", validJWTSecret)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("server port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Server.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("read header timeout = %s, want 5s", cfg.Server.ReadHeaderTimeout)
	}
	if cfg.Database.Host != "localhost" {
		t.Errorf("db host = %q, want localhost", cfg.Database.Host)
	}
	if cfg.Database.MaxConns != 25 {
		t.Errorf("db max conns = %d, want 25", cfg.Database.MaxConns)
	}
	if cfg.Log.Level != "info" || cfg.Log.Format != "json" {
		t.Errorf("log config = %+v, want info/json", cfg.Log)
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", validJWTSecret)
	t.Setenv("SERVER_PORT", "9999")
	t.Setenv("SERVER_ALLOWED_ORIGINS", "https://a.example,https://b.example")
	t.Setenv("DB_NAME", "custom")
	t.Setenv("LOG_FORMAT", "pretty")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("server port = %d, want 9999", cfg.Server.Port)
	}
	if got, want := len(cfg.Server.AllowedOrigins), 2; got != want {
		t.Fatalf("allowed origins = %v, want %d entries", cfg.Server.AllowedOrigins, want)
	}
	if cfg.Database.DBName != "custom" {
		t.Errorf("db name = %q, want custom", cfg.Database.DBName)
	}
	if cfg.Log.Format != "pretty" {
		t.Errorf("log format = %q, want pretty", cfg.Log.Format)
	}
}

func TestValidateRejectsPlaceholderSecret(t *testing.T) {
	cfg := validConfig()
	cfg.JWT.SecretKey = DefaultJWTSecret

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected an error for the placeholder secret")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET_KEY") {
		t.Errorf("error %q should mention JWT_SECRET_KEY", err)
	}
}

func TestValidateRejectsShortSecret(t *testing.T) {
	cfg := validConfig()
	cfg.JWT.SecretKey = "too-short"

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected an error for a short secret")
	}
}

func TestValidateReportsAllProblems(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Port = 0
	cfg.Database.Port = 70000
	cfg.Log.Level = "verbose"
	cfg.Log.Format = "xml"
	cfg.JWT.SecretKey = ""

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation errors")
	}

	for _, want := range []string{"server port", "database port", "log level", "log format", "JWT_SECRET_KEY"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}
}

func TestValidateRejectsInvertedExpiry(t *testing.T) {
	cfg := validConfig()
	cfg.JWT.AccessExpiry = 48 * time.Hour
	cfg.JWT.RefreshExpiry = time.Hour

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected an error for refresh expiry shorter than access expiry")
	}
}

func TestDSN(t *testing.T) {
	cfg := validConfig()
	cfg.Database.Host = "db.internal"
	cfg.Database.Port = 6432
	cfg.Database.User = "app"
	cfg.Database.Password = "p@ss word"
	cfg.Database.DBName = "svc"
	cfg.Database.SSLMode = "require"

	// A test fixture, not a real credential.
	const want = "postgres://app:p%40ss%20word@db.internal:6432/svc?sslmode=require" //nolint:gosec // test fixture
	if got := cfg.Database.DSN(); got != want {
		t.Errorf("DSN() = %q, want %q", got, want)
	}
}

func TestServerAddr(t *testing.T) {
	cfg := validConfig()
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 8081

	if got, want := cfg.Server.Addr(), "127.0.0.1:8081"; got != want {
		t.Errorf("Addr() = %q, want %q", got, want)
	}
}

// validConfig returns a configuration that passes Validate.
func validConfig() Config {
	return Config{
		Server: ServerConfig{
			Host:              "0.0.0.0",
			Port:              8080,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
			RequestTimeout:    30 * time.Second,
			ShutdownTimeout:   10 * time.Second,
			MaxBodyBytes:      1 << 20,
			AllowedOrigins:    []string{"*"},
		},
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			User:            "gotemplate",
			Password:        "gotemplate",
			DBName:          "gotemplate",
			SSLMode:         "disable",
			MaxConns:        25,
			MinConns:        5,
			MaxConnLifetime: 30 * time.Minute,
			MaxConnIdleTime: 5 * time.Minute,
		},
		JWT: JWTConfig{
			SecretKey:     validJWTSecret,
			AccessExpiry:  15 * time.Minute,
			RefreshExpiry: 168 * time.Hour,
		},
		Log: LogConfig{Level: "info", Format: "json"},
	}
}

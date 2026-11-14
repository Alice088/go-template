// Package config loads, validates and stores application configuration.
package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// DefaultJWTSecret is the placeholder secret used by .env.example.
// The application refuses to start with it, so a template copy cannot
// silently run in production with a known signing key.
const DefaultJWTSecret = "change-me-in-production"

// minJWTSecretLength is the minimum accepted length of the JWT signing key.
const minJWTSecretLength = 32

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig   `envPrefix:"SERVER_"`
	Database DatabaseConfig `envPrefix:"DB_"`
	JWT      JWTConfig      `envPrefix:"JWT_"`
	Log      LogConfig      `envPrefix:"LOG_"`
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Host              string        `env:"HOST" envDefault:"0.0.0.0"`
	Port              int           `env:"PORT" envDefault:"8080"`
	ReadHeaderTimeout time.Duration `env:"READ_HEADER_TIMEOUT" envDefault:"5s"`
	ReadTimeout       time.Duration `env:"READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout      time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
	IdleTimeout       time.Duration `env:"IDLE_TIMEOUT" envDefault:"60s"`
	RequestTimeout    time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
	ShutdownTimeout   time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	MaxBodyBytes      int64         `env:"MAX_BODY_BYTES" envDefault:"1048576"`
	AllowedOrigins    []string      `env:"ALLOWED_ORIGINS" envDefault:"*"`
}

// Addr returns the host:port address the HTTP server listens on.
func (c *ServerConfig) Addr() string {
	return net.JoinHostPort(c.Host, fmt.Sprint(c.Port))
}

// DatabaseConfig contains PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host            string        `env:"HOST" envDefault:"localhost"`
	Port            int           `env:"PORT" envDefault:"5432"`
	User            string        `env:"USER" envDefault:"postgres"`
	Password        string        `env:"PASSWORD" envDefault:"postgres"`
	DBName          string        `env:"NAME" envDefault:"gotemplate"`
	SSLMode         string        `env:"SSLMODE" envDefault:"disable"`
	MaxConns        int32         `env:"MAX_CONNS" envDefault:"25"`
	MinConns        int32         `env:"MIN_CONNS" envDefault:"5"`
	MaxConnLifetime time.Duration `env:"CONN_MAX_LIFETIME" envDefault:"30m"`
	MaxConnIdleTime time.Duration `env:"CONN_MAX_IDLE_TIME" envDefault:"5m"`
}

// DSN returns a PostgreSQL connection URL.
func (c *DatabaseConfig) DSN() string {
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     net.JoinHostPort(c.Host, fmt.Sprint(c.Port)),
		Path:     "/" + c.DBName,
		RawQuery: url.Values{"sslmode": []string{c.SSLMode}}.Encode(),
	}
	return u.String()
}

// JWTConfig contains JWT signing and expiry settings.
type JWTConfig struct {
	SecretKey     string        `env:"SECRET_KEY"`
	AccessExpiry  time.Duration `env:"ACCESS_EXPIRY" envDefault:"15m"`
	RefreshExpiry time.Duration `env:"REFRESH_EXPIRY" envDefault:"168h"`
}

// LogConfig contains logging level and format settings.
type LogConfig struct {
	Level  string `env:"LEVEL" envDefault:"info"`
	Format string `env:"FORMAT" envDefault:"json"`
}

// Log levels and formats understood by pkg logger setup.
var (
	validLogLevels  = []string{"debug", "info", "warn", "error"}
	validLogFormats = []string{"json", "pretty"}
)

// Load parses environment variables, validates the result and returns Config.
func Load() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("parse environment: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Validate checks that the configuration is safe to run with.
// All problems are reported at once.
func (c *Config) Validate() error {
	var problems []error

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		problems = append(problems, fmt.Errorf("server port %d is out of range 1-65535", c.Server.Port))
	}
	if c.Server.ReadHeaderTimeout <= 0 {
		problems = append(problems, errors.New("server read header timeout must be positive"))
	}
	if c.Server.ReadTimeout <= 0 {
		problems = append(problems, errors.New("server read timeout must be positive"))
	}
	if c.Server.WriteTimeout <= 0 {
		problems = append(problems, errors.New("server write timeout must be positive"))
	}
	if c.Server.IdleTimeout <= 0 {
		problems = append(problems, errors.New("server idle timeout must be positive"))
	}
	if c.Server.RequestTimeout <= 0 {
		problems = append(problems, errors.New("server request timeout must be positive"))
	}
	if c.Server.MaxBodyBytes <= 0 {
		problems = append(problems, errors.New("server max body bytes must be positive"))
	}
	for _, origin := range c.Server.AllowedOrigins {
		if strings.TrimSpace(origin) == "" {
			problems = append(problems, errors.New("server allowed origins must not contain empty values"))
			break
		}
	}

	if c.Database.Host == "" {
		problems = append(problems, errors.New("database host is required"))
	}
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		problems = append(problems, fmt.Errorf("database port %d is out of range 1-65535", c.Database.Port))
	}
	if c.Database.User == "" {
		problems = append(problems, errors.New("database user is required"))
	}
	if c.Database.DBName == "" {
		problems = append(problems, errors.New("database name is required"))
	}
	if c.Database.MaxConns < 1 {
		problems = append(problems, errors.New("database max conns must be at least 1"))
	}
	if c.Database.MinConns < 0 || c.Database.MinConns > c.Database.MaxConns {
		problems = append(problems, errors.New("database min conns must be between 0 and max conns"))
	}

	switch {
	case c.JWT.SecretKey == "":
		problems = append(problems, errors.New("JWT_SECRET_KEY is required"))
	case c.JWT.SecretKey == DefaultJWTSecret:
		problems = append(problems,
			errors.New(`JWT_SECRET_KEY is still the placeholder value, generate one with: openssl rand -hex 32`))
	case len(c.JWT.SecretKey) < minJWTSecretLength:
		problems = append(problems,
			fmt.Errorf("JWT_SECRET_KEY must be at least %d characters long", minJWTSecretLength))
	}
	if c.JWT.AccessExpiry <= 0 {
		problems = append(problems, errors.New("jwt access expiry must be positive"))
	}
	if c.JWT.RefreshExpiry <= 0 {
		problems = append(problems, errors.New("jwt refresh expiry must be positive"))
	}
	if c.JWT.RefreshExpiry < c.JWT.AccessExpiry {
		problems = append(problems, errors.New("jwt refresh expiry must not be shorter than access expiry"))
	}

	if !contains(validLogLevels, c.Log.Level) {
		problems = append(problems,
			fmt.Errorf("log level %q is not one of %s", c.Log.Level, strings.Join(validLogLevels, ", ")))
	}
	if !contains(validLogFormats, c.Log.Format) {
		problems = append(problems,
			fmt.Errorf("log format %q is not one of %s", c.Log.Format, strings.Join(validLogFormats, ", ")))
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid configuration:\n%w", errors.Join(problems...))
	}
	return nil
}

func contains(allowed []string, value string) bool {
	for _, item := range allowed {
		if item == value {
			return true
		}
	}
	return false
}

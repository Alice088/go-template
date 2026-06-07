package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Server   ServerConfig   `envPrefix:"SERVER_"`
	Database DatabaseConfig `envPrefix:"DB_"`
	JWT      JWTConfig      `envPrefix:"JWT_"`
	Log      LogConfig      `envPrefix:"LOG_"`
}

type ServerConfig struct {
	Host         string        `env:"HOST" envDefault:"0.0.0.0"`
	Port         int           `env:"PORT" envDefault:"8080"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" envDefault:"10s"`
}

type DatabaseConfig struct {
	Host     string `env:"HOST" envDefault:"localhost"`
	Port     int    `env:"PORT" envDefault:"5432"`
	User     string `env:"USER" envDefault:"postgres"`
	Password string `env:"PASSWORD" envDefault:"postgres"`
	DBName   string `env:"NAME" envDefault:"gotemplate"`
	SSLMode  string `env:"SSLMODE" envDefault:"disable"`
}

type JWTConfig struct {
	SecretKey     string        `env:"SECRET_KEY" envDefault:"change-me-in-production"`
	AccessExpiry  time.Duration `env:"ACCESS_EXPIRY" envDefault:"15m"`
	RefreshExpiry time.Duration `env:"REFRESH_EXPIRY" envDefault:"168h"`
}

type LogConfig struct {
	Level  string `env:"LEVEL" envDefault:"info"`
	Format string `env:"FORMAT" envDefault:"json"`
}

func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

func Load() (*Config, error) {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

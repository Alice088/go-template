// Package logger configures the zerolog global logger.
package logger

import (
	"go-template/internal/config"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Setup applies the log level and output format to the global logger.
func Setup(cfg config.LogConfig) {
	zerolog.SetGlobalLevel(parseLevel(cfg.Level))
	zerolog.TimeFieldFormat = time.RFC3339

	if cfg.Format == "pretty" {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		})
	} else {
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}

	log.Debug().Str("level", cfg.Level).Str("format", cfg.Format).Msg("logger configured")
}

// parseLevel maps a configuration value to a zerolog level.
// Invalid values fall back to info; config validation rejects them earlier.
func parseLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

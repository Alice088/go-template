package database

import (
	"context"
	"fmt"
	"go-template/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Postgres wraps a pgx connection pool.
type Postgres struct {
	Pool *pgxpool.Pool
}

// NewPostgres creates a new connection pool and verifies connectivity.
// Pool sizing and lifetimes come from the configuration.
func NewPostgres(ctx context.Context, cfg config.DatabaseConfig) (*Postgres, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	log.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.DBName).
		Int32("max_conns", cfg.MaxConns).
		Msg("connected to PostgreSQL")

	return &Postgres{Pool: pool}, nil
}

// Ping checks that the database is reachable.
func (p *Postgres) Ping(ctx context.Context) error {
	if p.Pool == nil {
		return fmt.Errorf("connection pool is not initialized")
	}
	return p.Pool.Ping(ctx)
}

// Close gracefully shuts down the connection pool.
func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
		log.Info().Msg("connection pool closed")
	}
}

// Package main Go Template API
//
// @title          Go Template API
// @version        0.1.0
// @description    A Go API project template
// @termsOfService https://github.com/Alice088/go-template
//
// @contact.name   Developer
// @contact.email  dev@example.com
//
// @license.name MIT
// @license.url  https://opensource.org/licenses/MIT
//
// @host      localhost:8080
// @BasePath  /v1
//
// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description               Type "Bearer " followed by a JWT token
package main

import (
	"context"
	"errors"
	"fmt"
	"go-template/internal/config"
	"go-template/internal/database"
	"go-template/internal/handler"
	"go-template/internal/logger"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	v1 "go-template/internal/router/v1"
)

// version is the application version. It is overridden at build time with
// -ldflags "-X main.version=...".
var version = "dev"

// databaseConnectTimeout bounds the initial database handshake.
const databaseConnectTimeout = 5 * time.Second

func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("application failed")
	}
}

// run wires the application together and blocks until it is shut down.
// Errors are returned instead of exiting, so deferred cleanups always run.
func run() error {
	if err := godotenv.Load(); err != nil {
		log.Warn().Err(err).Msg("no .env file found, using environment")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger.Setup(cfg.Log)
	log.Info().Str("version", version).Msg("starting application")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	connectCtx, cancelConnect := context.WithTimeout(ctx, databaseConnectTimeout)
	db, err := database.NewPostgres(connectCtx, cfg.Database)
	cancelConnect()
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()

	router := v1.NewRouter(handler.NewHealthHandler(version, db), cfg)

	srv := &http.Server{
		Addr:              cfg.Server.Addr(),
		Handler:           router,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info().Str("addr", srv.Addr).Msg("starting server")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("serve http: %w", err)
		}
		return nil
	case <-ctx.Done():
		log.Info().Msg("shutdown signal received")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	log.Info().Msg("server stopped gracefully")
	return nil
}

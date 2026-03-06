// Copyright (C) 2024 Portway Contributors
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//
// For commercial licensing, contact: licensing@portway.dev

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/portway/portway/internal/api"
	"github.com/portway/portway/internal/config"
	"github.com/portway/portway/internal/db"
)

func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("server exited with error")
	}
}

func run() error {
	// -- Configuration ----------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("main: %w", err)
	}

	// -- Logger -----------------------------------------------------------
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	var logger zerolog.Logger
	if cfg.Environment == "development" {
		logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	} else {
		logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}

	logger.Info().
		Str("environment", cfg.Environment).
		Str("log_level", level.String()).
		Msg("portway starting")

	// -- Database ---------------------------------------------------------
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("main: %w", err)
	}
	defer pool.Close()
	logger.Info().Msg("database pool ready")

	// -- HTTP Router ------------------------------------------------------
	router := api.NewRouter(api.RouterConfig{
		Logger: logger,
	})

	// -- HTTP Server ------------------------------------------------------
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info().Str("addr", srv.Addr).Msg("HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	<-quit
	logger.Info().Msg("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("main: graceful shutdown failed: %w", err)
	}

	logger.Info().Msg("server stopped cleanly")
	return nil
}

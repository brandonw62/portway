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
	"fmt"
	"os"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/portway/portway/internal/config"
	"github.com/portway/portway/internal/db"
	"github.com/portway/portway/internal/jobs"
)

func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("worker exited with error")
	}
}

func run() error {
	// -- Configuration ----------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("worker: %w", err)
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

	logger.Info().Str("environment", cfg.Environment).Msg("portway worker starting")

	// -- Database ---------------------------------------------------------
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("worker: %w", err)
	}
	defer pool.Close()
	logger.Info().Msg("database pool ready")

	// Suppress the pool variable until it is wired into handlers.
	_ = pool

	// -- Valkey / Asynq ---------------------------------------------------
	redisOpt, err := asynq.ParseRedisURI(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("worker: failed to parse redis URL: %w", err)
	}

	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"critical": 6,
			"default":  3,
			"low":      1,
		},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			logger.Error().
				Err(err).
				Str("task_type", task.Type()).
				Msg("task failed")
		}),
	})

	// -- Task Router ------------------------------------------------------
	mux := asynq.NewServeMux()

	// Register handlers here as they are implemented.
	// Example: mux.HandleFunc(jobs.TypeGitHubSync, githubHandler.HandleGitHubSync)
	mux.HandleFunc(jobs.TypeGitHubSync, func(ctx context.Context, t *asynq.Task) error {
		logger.Info().Str("task_type", t.Type()).Msg("received task (no-op placeholder)")
		return nil
	})

	// -- Start ------------------------------------------------------------
	logger.Info().Msg("worker listening for tasks")
	if err := srv.Run(mux); err != nil {
		return fmt.Errorf("worker: asynq server failed: %w", err)
	}

	return nil
}

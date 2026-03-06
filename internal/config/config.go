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

package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config holds all runtime configuration sourced from environment variables.
type Config struct {
	DatabaseURL          string `env:"DATABASE_URL,required"`
	RedisURL             string `env:"REDIS_URL,required"`
	ServerPort           int    `env:"SERVER_PORT" envDefault:"8080"`
	GitHubAppID          int64  `env:"GITHUB_APP_ID"`
	GitHubPrivateKeyPath string `env:"GITHUB_PRIVATE_KEY_PATH"`
	GitHubWebhookSecret  string `env:"GITHUB_WEBHOOK_SECRET"`
	LogLevel             string `env:"LOG_LEVEL" envDefault:"info"`
	Environment          string `env:"ENVIRONMENT" envDefault:"development"`
}

// Load reads configuration from environment variables and returns a Config.
// Returns an error if any required variables are missing or malformed.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config: failed to parse environment: %w", err)
	}
	return cfg, nil
}

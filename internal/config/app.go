package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert/yaml"
)

type Config struct {
	BadgerDir          string `yaml:"badger_dir"`
	DownNodesRmIvlMSec int    `yaml:"down_nodes_rm_ivl_msec"`
	HealthcheckIvlMsec int    `yaml:"healthcheck_ivl_msec"`
	BaseURL            string `yaml:"base_url"`

	APIKey string `env:"SERVICE_DISCOVERY_API_KEY"`
}

func New(logger zerolog.Logger) (*Config, error) {
	cfg := Config{
		BadgerDir:          "./badger",
		DownNodesRmIvlMSec: 3_000,
		HealthcheckIvlMsec: 1_000,
		BaseURL:            "0.0.0.0:6500",
	}

	if err := godotenv.Load(); err != nil {
		logger.
			Warn().
			Msg("No .env file detected")
	}

	data, err := os.ReadFile("config.yaml")
	if err != nil {
		logger.
			Warn().
			Err(fmt.Errorf("reading file: %w", err)).
			Msg("Using defaults")
	}

	cfg.APIKey = os.Getenv("SERVICE_DISCOVERY_API_KEY")
	if cfg.APIKey == "" {
		return nil, errors.New("missing SERVICE_DISCOVERY_API_KEY env")
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling yaml: %w", err)
	}

	return &cfg, nil
}

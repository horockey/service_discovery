package config

type Config struct {
	BadgerDir          string `yaml:"badger_dir"`
	DownNodesRmIvlMSec int    `yaml:"down_nodes_rm_ivl_msec"`
	HealthcheckIvlMsec int    `yaml:"healthcheck_ivl_msec"`
	BaseURL            string `yaml:"base_url"`

	APIKey string `env:"SERVICE_DISCOVERY_API_KEY"`
}

// TODO: impl
func New() (*Config, error)

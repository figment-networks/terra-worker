package config

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"github.com/kelseyhightower/envconfig"
)

var (
	Name      = "terra-worker"
	Version   string
	GitSHA    string
	Timestamp string
)

const (
	modeDevelopment = "development"
	modeProduction  = "production"
)

// Config holds the configuration data
type Config struct {
	AppEnv string `json:"app_env" envconfig:"APP_ENV" default:"development"`

	Address  string `json:"address" envconfig:"ADDRESS" default:"0.0.0.0"`
	Port     string `json:"port" envconfig:"PORT" default:"3000"`
	HTTPPort string `json:"http_port" envconfig:"HTTP_PORT" default:"8087"`

	Managers        string        `json:"managers" envconfig:"MANAGERS" default:"127.0.0.1:8085"`
	ManagerInterval time.Duration `json:"manager_interval" envconfig:"MANAGER_INTERVAL" default:"10s"`
	Hostname        string        `json:"hostname" envconfig:"HOSTNAME"`

	TerraGRPCAddr string `json:"terra_grpc_addr" envconfig:"TERRA_GRPC_ADDR" required:"true"`
	DatahubKey    string `json:"datahub_key" envconfig:"DATAHUB_KEY"`
	ChainID       string `json:"chain_id" envconfig:"CHAIN_ID"`

	MaximumHeightsToGet float64 `json:"maximum_heights_to_get" envconfig:"MAXIMUM_HEIGHTS_TO_GET" default:"10000"`
	RequestsPerSecond   int64   `json:"requests_per_second" envconfig:"REQUESTS_PER_SECOND" default:"33"`

	StoreHTTPEndpoints  string        `json:"store_http_endpoints" envconfig:"STORE_HTTP_ENDPOINTS"`
	HealthCheckInterval time.Duration `json:"health_check_interval" envconfig:"HEALTH_CHECK_INTERVAL" default:"10s"`

	// Rollbar
	RollbarAccessToken string `json:"rollbar_access_token" envconfig:"ROLLBAR_ACCESS_TOKEN"`
	RollbarServerRoot  string `json:"rollbar_server_root" envconfig:"ROLLBAR_SERVER_ROOT" default:"github.com/figment-networks/terra-worker"`

	TimeoutBlockCall       time.Duration `json:"timeout_block_call" envconfig:"TIMEOUT_BLOCK_CALL" default:"30s"`
	TimeoutTransactionCall time.Duration `json:"timeout_transaction_call" envconfig:"TIMEOUT_TRANSACTION_CALL" default:"30s"`
}

// FromFile reads the config from a file
func FromFile(path string, config *Config) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, config)
}

// FromEnv reads the config from environment variables
func FromEnv(config *Config) error {
	return envconfig.Process("", config)
}

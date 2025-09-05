package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	AppName  string `env:"APP_NAME" envDefault:"network-topology-svc"`
	AppEnv   string `env:"APP_ENV"  envDefault:"dev"`
	HTTPAddr string `env:"HTTP_ADDR" envDefault:":8080"`

	// Auth
	AuthBearerToken string `env:"AUTH_BEARER_TOKEN" envDefault:""` // optional; if empty, auth is disabled

	// Logging
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	LogJSON     bool   `env:"LOG_JSON" envDefault:"true"`
	LogFilePath string `env:"LOG_FILE_PATH" envDefault:"./logs/app.log"`

	// Postgres
	PostgresDSN string `env:"POSTGRES_DSN,required"` // e.g. postgres://user:pass@host:5432/dbname?sslmode=disable
	PGMaxConns  int    `env:"PG_MAX_CONNS" envDefault:"10"`

	// Redis (placeholder; not required by Phase 1)
	RedisURL string `env:"REDIS_URL" envDefault:""`

	// Topology window and drift (Phase 1)
	TopologyWindow time.Duration `env:"TOPOLOGY_WINDOW" envDefault:"1h"`
	DriftAllowance time.Duration `env:"DRIFT_ALLOWANCE" envDefault:"2m"`
}

func Load() (Config, error) {
	var c Config
	return c, env.Parse(&c)
}

package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	AppName     string        `env:"APP_NAME" envDefault:"network-topology-service"`
	AppEnv      string        `env:"APP_ENV" envDefault:"dev"`
	HTTPPort    int           `env:"HTTP_PORT" envDefault:"8080"`
	HTTPReadTO  time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"15s"`
	HTTPWriteTO time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"15s"`

	APIKey string `env:"API_KEY" envDefault:"dev-secret-key"`

	PGHost        string        `env:"PG_HOST" envDefault:"postgres"`
	PGPort        int           `env:"PG_PORT" envDefault:"5432"`
	PGUser        string        `env:"PG_USER" envDefault:"app"`
	PGPassword    string        `env:"PG_PASSWORD" envDefault:"app"`
	PGDatabase    string        `env:"PG_DATABASE" envDefault:"topology"`
	PGSSLMode     string        `env:"PG_SSLMODE" envDefault:"disable"`
	PGMaxConns    int32         `env:"PG_MAX_CONNS" envDefault:"10"`
	PGMinConns    int32         `env:"PG_MIN_CONNS" envDefault:"1"`
	PGMaxLifetime time.Duration `env:"PG_MAX_CONN_LIFETIME" envDefault:"30m"`

	TopologyWindow time.Duration `env:"TOPOLOGY_WINDOW" envDefault:"1h"`
	TopologyDrift  time.Duration `env:"TOPOLOGY_DRIFT" envDefault:"2m"`

	LogPath       string `env:"LOG_PATH" envDefault:"/var/log/app/app.log"`
	LogMaxSizeMB  int    `env:"LOG_MAX_SIZE_MB" envDefault:"50"`
	LogMaxBackups int    `env:"LOG_MAX_BACKUPS" envDefault:"5"`
	LogMaxAgeDays int    `env:"LOG_MAX_AGE_DAYS" envDefault:"30"`
	LogLevel      string `env:"LOG_LEVEL" envDefault:"info"`
	TLS_CERT      string `env:"TLS_CERT"`
	TLS_KEY       string `env:"TLS_KEY"`
}

func Load() (*Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

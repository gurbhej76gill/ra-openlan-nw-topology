package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type TopicConfig struct {
	MinBytes       int           `yaml:"min_bytes"`
	MaxBytes       int           `yaml:"max_bytes"`
	CommitInterval time.Duration `yaml:"commit_interval"`
	Concurrency    int           `yaml:"concurrency"`
}

type Config struct {
	AppName     string        `env:"APP_NAME" envDefault:""`
	AppEnv      string        `env:"APP_ENV" envDefault:"dev"`
	HTTPPort    int           `env:"HTTP_PORT" envDefault:"8088"`
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

	TopologyWindow          time.Duration `env:"TOPOLOGY_WINDOW" envDefault:"1h"`
	TopologyDrift           time.Duration `env:"TOPOLOGY_DRIFT" envDefault:"2m"`
	TimepointsAPIBaseURL    string        `env:"TIMEPOINTS_API_BASE_URL" envDefault:"https://localhost:16009"`
	TimepointsAPIMaxRecords int           `env:"TIMEPOINTS_API_MAX_RECORDS" envDefault:"100"`
	TimepointsAPIToken      string        `env:"TIMEPOINTS_API_TOKEN" envDefault:""`
	TimepointsServiceType   string        `env:"TIMEPOINTS_SERVICE_TYPE" envDefault:"timepoints-api"`

	// kafka
	KafkaBrokers         []string      `env:"NW_KAFKA_BROKERS" envSeparator:","`
	KafkaTopicCmd        string        `env:"NW_KAFKA_TOPIC_CMD" envDefault:"CnC"`
	KafkaTopicResp       string        `env:"NW_KAFKA_TOPIC_RESP" envDefault:"CnC_Res"`
	KafkaTopics          []string      `yaml:"NW_KAFKA_TOPICS" envDefault:"service_events"`
	KafkaGroupID         string        `env:"NW_KAFKA_GROUP_ID" envDefault:"cgw-wrapper"`
	KafkaDialTimeout     time.Duration `env:"NW_KAFKA_DIAL_TIMEOUT" envDefault:"5s"`
	KafkaWriteTimeout    time.Duration `env:"NW_KAFKA_WRITE_TIMEOUT" envDefault:"15s"`
	KafkaReadTimeout     time.Duration `env:"NW_KAFKA_READ_TIMEOUT" envDefault:"5s"`
	KafkaMinBytes        int           `env:"NW_KAFKA_MIN_BYTES" envDefault:"1"`
	KafkaMaxBytes        int           `env:"NW_KAFKA_MAX_BYTES" envDefault:"1048576"`
	KafkaAllowAutoCreate bool          `env:"NW_KAFKA_ALLOW_AUTO_CREATE" envDefault:"true"`
	KafkaTopicLifecycle  string        `env:"NW_KAFKA_TOPIC_LIFECYCLE" envDefault:"service_events"`

	LogPath               string `env:"LOG_PATH" envDefault:"/var/log/app/app.log"`
	LogMaxSizeMB          int    `env:"LOG_MAX_SIZE_MB" envDefault:"50"`
	LogMaxBackups         int    `env:"LOG_MAX_BACKUPS" envDefault:"5"`
	LogMaxAgeDays         int    `env:"LOG_MAX_AGE_DAYS" envDefault:"30"`
	LogLevel              string `env:"LOG_LEVEL" envDefault:"info"`
	LogFile               string `env:"NW_LOG_FILE" envDefault:"cgw-wrapper.log"`
	LogJSON               bool   `env:"NW_LOG_JSON" envDefault:"false"`
	TLS_CERT              string `env:"TLS_CERT"`
	TLS_KEY               string `env:"TLS_KEY"`
	TokenValidationCACert string `env:"TOKEN_VALIDATION_CA_CERT"`

	// lifecycle event config
	PrivateEndpoint   string        `env:"NW_PRIVATE_ENDPOINT"`
	PublicEndpoint    string        `env:"NW_PUBLIC_ENDPOINT"`
	ServiceType       string        `env:"NW_SERVICE_TYPE" envDefault:"cgw-rest-2"`
	LifecycleInterval time.Duration `env:"NW_LIFECYCLE_INTERVAL" envDefault:"5s"`
	BuildVersion      string        `env:"NW_BUILD_VERSION" envDefault:"dev"`
	RequestTimeout    time.Duration `env:"NW_REQUEST_TIMEOUT" envDefault:"30s"`
}

func Load() (*Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

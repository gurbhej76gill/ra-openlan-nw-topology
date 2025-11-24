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

	PGHost        string        `env:"STORAGE_TYPE_POSTGRESQL_HOST" envDefault:"postgres"`
	PGPort        int           `env:"STORAGE_TYPE_POSTGRESQL_PORT" envDefault:"5432"`
	PGUser        string        `env:"STORAGE_TYPE_POSTGRESQL_USERNAME" envDefault:"app"`
	PGPassword    string        `env:"STORAGE_TYPE_POSTGRESQL_PASSWORD" envDefault:"app"`
	PGDatabase    string        `env:"STORAGE_TYPE_POSTGRESQL_DATABASE" envDefault:"topology"`
	PGSSLMode     string        `env:"STORAGE_TYPE_POSTGRESQL_SSLMODE" envDefault:"disable"`
	PGMaxConns    int32         `env:"STORAGE_TYPE_POSTGRESQL_MAX_CONNS" envDefault:"10"`
	PGMinConns    int32         `env:"STORAGE_TYPE_POSTGRESQL_MIN_CONNS" envDefault:"1"`
	PGMaxLifetime time.Duration `env:"STORAGE_TYPE_POSTGRESQL_MAX_CONN_LIFETIME" envDefault:"30m"`

	TopologyWindow time.Duration `env:"TOPOLOGY_WINDOW" envDefault:"1h"`
	TopologyDrift  time.Duration `env:"TOPOLOGY_DRIFT" envDefault:"2m"`
	// kafka
	KafkaBrokers         []string      `env:"KAFKA_BROKERS" envSeparator:","`
	KafkaTopicCmd        string        `env:"KAFKA_TOPIC_CMD" envDefault:"CnC"`
	KafkaTopicResp       string        `env:"KAFKA_TOPIC_RESP" envDefault:"CnC_Res"`
	KafkaTopics          []string      `yaml:"KAFKA_TOPICS" envDefault:"service_events"`
	KafkaGroupID         string        `env:"KAFKA_GROUP_ID" envDefault:"nwtopology-service-group"`
	KafkaDialTimeout     time.Duration `env:"KAFKA_DIAL_TIMEOUT" envDefault:"5s"`
	KafkaWriteTimeout    time.Duration `env:"KAFKA_WRITE_TIMEOUT" envDefault:"15s"`
	KafkaReadTimeout     time.Duration `env:"KAFKA_READ_TIMEOUT" envDefault:"5s"`
	KafkaMinBytes        int           `env:"KAFKA_MIN_BYTES" envDefault:"1"`
	KafkaMaxBytes        int           `env:"KAFKA_MAX_BYTES" envDefault:"1048576"`
	KafkaAllowAutoCreate bool          `env:"KAFKA_ALLOW_AUTO_CREATE" envDefault:"true"`
	KafkaTopicLifecycle  string        `env:"KAFKA_TOPIC_LIFECYCLE" envDefault:"service_events"`

	LogPath               string `env:"SYSTEM_LOG_PATH" envDefault:"/var/log/app/app.log"`
	LogMaxSizeMB          int    `env:"SYSTEM_LOG_MAX_SIZE_MB" envDefault:"50"`
	LogMaxBackups         int    `env:"SYSTEM_LOG_MAX_BACKUPS" envDefault:"5"`
	LogMaxAgeDays         int    `env:"SYSTEM_LOG_MAX_AGE_DAYS" envDefault:"30"`
	LogLevel              string `env:"SYSTEM_LOG_LEVEL" envDefault:"info"`
	LogFile               string `env:"SYSTEM__LOG_FILE" envDefault:"nwtopology.log"`
	LogJSON               bool   `env:"SYSTEM__LOG_JSON" envDefault:"false"`
	TLS_CERT              string `env:"INTERNAL_RESTAPI_HOST_CERT"`
	TLS_KEY               string `env:"INTERNAL_RESTAPI_HOST_KEY"`
	TokenValidationCACert string `env:"INTERNAL_RESTAPI_HOST_ROOTCA"`

	// lifecycle event config
	PrivateEndpoint   string        `env:"SYSTEM_URI_PRIVATE"`
	PublicEndpoint    string        `env:"SYSTEM_URI_PUBLIC"`
	ServiceType       string        `env:"SERVICE_TYPE" envDefault:"nwtopology"`
	LifecycleInterval time.Duration `env:"LIFECYCLE_INTERVAL" envDefault:"5s"`
	BuildVersion      string        `env:"BUILD_VERSION" envDefault:"dev"`
	RequestTimeout    time.Duration `env:"REQUEST_TIMEOUT" envDefault:"30s"`
}

func Load() (*Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

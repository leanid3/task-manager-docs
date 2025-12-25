package config

import (
	"log/slog"
	"time"

	"app/pkg/database/connector/sql/postgres"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Logger   LoggerConfig    `mapstructure:"logger" yaml:"logger"`
	Server   ServerConfig    `mapstructure:"server" yaml:"server"`
	Database postgres.Config `mapstructure:"database" yaml:"database"`
	Broker   BrokerConfig    `mapstructure:"broker" yaml:"broker"`
	Metrics  MetricsConfig   `mapstructure:"metrics" yaml:"metrics"`
	Minio    MinioConfig     `mapstructure:"minio" yaml:"minio"`
	Swagger  SwaggerConfig   `mapstructure:"swagger" yaml:"swagger"`
}

type BrokerConfig struct {
	// Общие настройки
	BootstrapService string   `mapstructure:"bootstrap_service" validate:"required" yaml:"bootstrap_service" env:"BROKER_BOOTSTRAP_SERVICE"`
	ClientID         string   `mapstructure:"client_id" validate:"required" yaml:"client_id" env:"BROKER_CLIENT_ID"`
	Topics           []string `mapstructure:"topics" yaml:"topics" env:"BROKER_TOPICS"`

	// Producer настройки
	ProducerAcks              string `mapstructure:"producer_acks" validate:"oneof=-1 all 0 1" yaml:"producer_acks" env:"BROKER_PRODUCER_ACKS"`
	ProducerEnableIdempotence bool   `mapstructure:"producer_idempotence" yaml:"producer_idempotence" env:"BROKER_PRODUCER_IDEMPOTENCE"`
	ProducerCompressionType   string `mapstructure:"producer_compression" validate:"oneof=snappy gzip lz4 zstd none" yaml:"producer_compression" env:"BROKER_PRODUCER_COMPRESSION"`
	ProducerRetries           int    `mapstructure:"producer_retries" yaml:"producer_retries" env:"BROKER_PRODUCER_RETRIES"`
	// ProducerBatchSize                  int    `mapstructure:"producer_batch_size"`
	// ProducerLingerMs                   int    `mapstructure:"producer_linger_ms"`
	// ProducerMaxInFlightRequestsPerConn int    `mapstructure:"producer_max_flight_requests"`
	// ProducerRequestTimeoutMs           int    `mapstructure:"producer_timeout_ms"`

	// Consumer настройки
	ConsumerGroupID              string `mapstructure:"consumer_group_id" yaml:"consumer_group_id" env:"BROKER_CONSUMER_GROUP_ID"`
	ConsumerEnableAutoCommit     bool   `mapstructure:"consumer_auto_commit" yaml:"consumer_auto_commit" env:"BROKER_CONSUMER_AUTO_COMMIT"`
	ConsumerAutoCommitIntervalMs int    `mapstructure:"consumer_commit_interval_ms" yaml:"consumer_commit_interval_ms" env:"BROKER_CONSUMER_COMMIT_INTERVAL_MS"`
	ConsumerSessionTimeoutMs     int    `mapstructure:"consumer_session_timeout_ms" yaml:"consumer_session_timeout_ms" env:"BROKER_CONSUMER_SESSION_TIMEOUT_MS"`
	ConsumerHeartbeatIntervalMs  int    `mapstructure:"consumer_heartbeat_interval_ms" yaml:"consumer_heartbeat_interval_ms" env:"BROKER_CONSUMER_HEARTBEAT_INTERVAL_MS"	`
	ConsumerMaxPollRecords       int    `mapstructure:"consumer_max_poll_records" yaml:"consumer_max_poll_records" env:"BROKER_CONSUMER_MAX_POLL_RECORDS"`

	// Безопасность (общая)
	SecurityProtocol string `mapstructure:"security_protocol" validate:"oneof=PLAINTEXT SASL_SSL SASL_PLAINTEXT SSL" yaml:"security_protocol" env:"BROKER_SECURITY_PROTOCOL"`
	SaslMechanism    string `mapstructure:"sasl_mechanism" validate:"omitempty,oneof=PLAIN SCRAM-SHA-256 SCRAM-SHA-512"`
	SaslUsername     string `mapstructure:"sasl_username" yaml:"sasl_username" env:"BROKER_SASL_USERNAME"`
	SaslPassword     string `mapstructure:"sasl_password" yaml:"sasl_password" env:"BROKER_SASL_PASSWORD"`

	// SSL
	SSLCertificateVerification bool `mapstructure:"ssl_certificate_verification" yaml:"ssl_certificate_verification" env:"BROKER_SSL_CERTIFICATE_VERIFICATION"`
}

type LoggerConfig struct {
	Level        string `mapstructure:"level" validate:"oneof=debug info warn error fatal" yaml:"level" env:"LOGGER_LEVEL"`
	Format       string `mapstructure:"format" validate:"oneof=text json"`
	Mode         string `mapstructure:"Mode" yaml:"mode" env:"LOGGER_MODE"`
	AppPath      string `mapstructure:"app_path" yaml:"app_path" env:"LOGGER_APP_PATH"`
	HealthPath   string `mapstructure:"health_path" yaml:"health_path" env:"LOGGER_HEALTH_PATH"`
	HTTPPath     string `mapstructure:"http_path" yaml:"http_path" env:"LOGGER_HTTP_PATH"`
	KafkaPath    string `mapstructure:"kafka_path" yaml:"kafka_path" env:"LOGGER_KAFKA_PATH"`
	MinioPath    string `mapstructure:"minio_path" yaml:"minio_path" env:"LOGGER_MINIO_PATH"	`
	TaskPath     string `mapstructure:"task_path" yaml:"task_path" env:"LOGGER_TASK_PATH"`
	S3Path       string `mapstructure:"s3_path" yaml:"s3_path" env:"LOGGER_S3_PATH"`
	MaxSize      int64  `mapstructure:"max_size" yaml:"max_size" env:"LOGGER_MAX_SIZE"`                   // максимальный размер файла в мегабайтах (0 = без ограничений)
	MaxFiles     int    `mapstructure:"max_files" yaml:"max_files" env:"LOGGER_MAX_FILES"`                // максимальное количество файлов (0 = без ограничений)
	ClearOnStart bool   `mapstructure:"clear_on_start" yaml:"clear_on_start" env:"LOGGER_CLEAR_ON_START"` // очищать ли файл при старте
}

type ServerConfig struct {
	Port            int           `mapstructure:"port" yaml:"port" env:"SERVER_PORT"`
	Host            string        `mapstructure:"host" yaml:"host" env:"SERVER_HOST"`
	Prefork         bool          `mapstructure:"prefork" yaml:"prefork" env:"SERVER_PREFORK"`
	Timeout         time.Duration `mapstructure:"timeout" yaml:"timeout" env:"SERVER_TIMEOUT"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout" yaml:"read_timeout" env:"SERVER_READ_TIMEOUT"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout" yaml:"write_timeout" env:"SERVER_WRITE_TIMEOUT"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" yaml:"shutdown_timeout" env:"SERVER_SHUTDOWN_TIMEOUT"`
}

type SwaggerConfig struct {
	Enabled bool   `mapstructure:"enabled" yaml:"enabled" env:"SWAGGER_ENABLED"`
	Path    string `mapstructure:"path" yaml:"path" env:"SWAGGER_PATH"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled" yaml:"enabled" env:"METRICS_ENABLED"`
	Port    int    `mapstructure:"port" yaml:"port" env:"METRICS_PORT"`
	Host    string `mapstructure:"host" yaml:"host" env:"METRICS_HOST"`
	Path    string `mapstructure:"path" yaml:"path" env:"METRICS_PATH"`
}

type MinioConfig struct {
	Endpoint  string        `mapstructure:"endpoint" yaml:"endpoint" env:"MINIO_ENDPOINT"`
	AccessKey string        `mapstructure:"access_key" yaml:"access_key" env:"MINIO_ACCESS_KEY"`
	SecretKey string        `mapstructure:"secret_key" yaml:"secret_key" env:"MINIO_SECRET_KEY"`
	Bucket    string        `mapstructure:"bucket" yaml:"bucket" env:"MINIO_BUCKET"`
	Prefix    string        `mapstructure:"prefix" yaml:"prefix" env:"MINIO_PREFIX"`
	UseSSL    bool          `mapstructure:"UseSSL" yaml:"UseSSL" env:"MINIO_USESSL"`
	Region    string        `mapstructure:"Region" yaml:"Region" env:"MINIO_REGION"`
	Timeout   time.Duration `mapstructure:"Timeout" yaml:"Timeout" env:"MINIO_TIMEOUT"`
}

func Load() (*Config, error) {
	var cfg Config

	// Сначала пытаемся загрузить из файла
	if err := cleanenv.ReadConfig("config.yaml", &cfg); err != nil {
		// Если файла нет, загружаем только из env
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, err
		}
		slog.Info("Config loaded from environment variables only")
	} else {
		slog.Info("Config loaded from file and environment variables")
	}

	slog.Info("CONFIG: ", "cfg", cfg)

	return &cfg, nil
}

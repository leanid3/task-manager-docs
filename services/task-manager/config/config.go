package config

import (
	"log/slog"
	"strings"
	"time"

	"app/pkg/database/connector/sql/postgres"

	"github.com/spf13/viper"
)

type Config struct {
	Logger   LoggerConfig    `mapstructure:"logger"`
	Server   ServerConfig    `mapstructure:"server"`
	Database postgres.Config `mapstructure:"database"`
	Broker   BrokerConfig    `mapstructure:"broker"`
	Metrics  MetricsConfig   `mapstructure:"metrics"`
	Minio    MinioConfig     `mapstructure:"minio"`
	Swagger  SwaggerConfig   `mapstructure:"swagger"`
}

type BrokerConfig struct {
	// Общие настройки
	BootstrapService string   `mapstructure:"bootstrap_service" validate:"required"`
	ClientID         string   `mapstructure:"client_id" validate:"required"`
	Topics           []string `mapstructure:"topics"`

	// Producer настройки
	ProducerAcks              string `mapstructure:"producer_acks" validate:"oneof=-1 all 0 1"`
	ProducerEnableIdempotence bool   `mapstructure:"producer_idempotence"`
	ProducerCompressionType   string `mapstructure:"producer_compression" validate:"oneof=snappy gzip lz4 zstd none"`
	ProducerRetries           int    `mapstructure:"producer_retries"`
	// ProducerBatchSize                  int    `mapstructure:"producer_batch_size"`
	// ProducerLingerMs                   int    `mapstructure:"producer_linger_ms"`
	// ProducerMaxInFlightRequestsPerConn int    `mapstructure:"producer_max_flight_requests"`
	// ProducerRequestTimeoutMs           int    `mapstructure:"producer_timeout_ms"`

	// Consumer настройки
	ConsumerGroupID              string `mapstructure:"consumer_group_id"`
	ConsumerEnableAutoCommit     bool   `mapstructure:"consumer_auto_commit"`
	ConsumerAutoCommitIntervalMs int    `mapstructure:"consumer_commit_interval_ms"`
	ConsumerSessionTimeoutMs     int    `mapstructure:"consumer_session_timeout_ms"`
	ConsumerHeartbeatIntervalMs  int    `mapstructure:"consumer_heartbeat_interval_ms"`
	// ConsumerMaxPollRecords       int    `mapstructure:"consumer_max_poll_records"`

	// Безопасность (общая)
	SecurityProtocol string `mapstructure:"security_protocol" validate:"oneof=PLAINTEXT SASL_SSL SASL_PLAINTEXT SSL"`
	SaslMechanism    string `mapstructure:"sasl_mechanism" validate:"omitempty,oneof=PLAIN SCRAM-SHA-256 SCRAM-SHA-512"`
	SaslUsername     string `mapstructure:"sasl_username"`
	SaslPassword     string `mapstructure:"sasl_password"`

	// SSL
	SSLCertificateVerification bool `mapstructure:"ssl_certificate_verification"`
}

type LoggerConfig struct {
	Level        string `mapstructure:"level" validate:"oneof=debug info warn error fatal"`
	Format       string `mapstructure:"format" validate:"oneof=text json"`
	Mode         string `mapstructure:"Mode"`
	AppPath      string `mapstructure:"app_path"`
	HealthPath   string `mapstructure:"health_path"`
	HTTPPath     string `mapstructure:"http_path"`
	KafkaPath    string `mapstructure:"kafka_path"`
	MinioPath    string `mapstructure:"minio_path"`
	TaskPath     string `mapstructure:"task_path"`
	S3Path       string `mapstructure:"s3_path"`
	MaxSize      int64  `mapstructure:"max_size"`       // максимальный размер файла в мегабайтах (0 = без ограничений)
	MaxFiles     int    `mapstructure:"max_files"`      // максимальное количество файлов (0 = без ограничений)
	ClearOnStart bool   `mapstructure:"clear_on_start"` // очищать ли файл при старте
}

type ServerConfig struct {
	Port            int           `mapstructure:"port"`
	Host            string        `mapstructure:"host"`
	Prefork         bool          `mapstructure:"prefork"`
	Timeout         time.Duration `mapstructure:"timeout"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

type SwaggerConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Host    string `mapstructure:"host"`
	Path    string `mapstructure:"path"`
}

type MinioConfig struct {
	Endpoint  string        `mapstructure:"endpoint"`
	AccessKey string        `mapstructure:"access_key"`
	SecretKey string        `mapstructure:"secret_key"`
	Bucket    string        `mapstructure:"bucket"`
	Prefix    string        `mapstructure:"prefix"`
	UseSSL    bool          `mapstructure:"UseSSL"`
	Region    string        `mapstructure:"Region"`
	Timeout   time.Duration `mapstructure:"Timeout"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/app")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Пытаемся прочитать конфигурационный файл, но не падаем, если его нет
	// В dev режиме используются только переменные окружения
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			slog.Info("Config file not found, using environment variables only")
		} else {
			slog.Warn("Error reading config file", "error", err)
		}
	} else {
		slog.Info("Config file loaded", "file", viper.ConfigFileUsed())
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

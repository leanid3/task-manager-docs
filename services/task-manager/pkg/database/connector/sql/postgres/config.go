package postgres

import "time"

type Config struct {
	Host            string        `mapstructure:"host" yaml:"host" env:"DATABASE_HOST"`
	Port            int           `mapstructure:"port" yaml:"port" env:"DATABASE_PORT"`
	User            string        `mapstructure:"user" yaml:"user" env:"DATABASE_USER"`
	Password        string        `mapstructure:"password" yaml:"password" env:"DATABASE_PASSWORD"`
	Database        string        `mapstructure:"database" yaml:"database" env:"DATABASE_DATABASE"`
	MaxConnections  int32         `mapstructure:"max_connections" yaml:"max_connections" env:"DATABASE_MAX_CONNECTIONS"`
	MinConnections  int32         `mapstructure:"min_connections" yaml:"min_connections" env:"DATABASE_MIN_CONNECTIONS"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime" yaml:"max_conn_lifetime" env:"DATABASE_MAX_CONN_LIFETIME"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time" yaml:"max_conn_idle_time" env:"DATABASE_MAX_CONN_IDLE_TIME"`
}

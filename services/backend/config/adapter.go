package config

import "app/pkg/database/connector/sql/postgres"

func (d *DatabaseConfig) ToPostgresConfig() *postgres.Config {
	return &postgres.Config{
		Host:            d.Host,
		Port:            d.Port,
		User:            d.User,
		Password:        d.Password,
		Database:        d.Database,
		MaxConnections:  d.MaxConnections,
		MinConnections:  d.MinConnections,
		MaxConnLifetime: d.MaxConnLifetime,
		MaxConnIdleTime: d.MaxConnIdleTime,
	}
}

package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	MaxConnections  int32
	MinConnections  int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

type Connector struct {
	pool *pgxpool.Pool
	cfg  *Config
}

func NewConnector(cfg *Config) (*Connector, error) {
	slog.Info("creating postgres connector", "config", cfg)
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?pool_max_conns=%d&pool_min_conns=%d",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.MaxConnections,
		cfg.MinConnections,
	)
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		slog.Error("failed to parse connection string", "error", err)
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Настраиваем пул
	poolConfig.MaxConns = cfg.MaxConnections
	poolConfig.MinConns = cfg.MinConnections
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		slog.Error("failed to create pool", "error", err)
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Проверяем, что соединение работает
	if err := pool.Ping(context.Background()); err != nil {
		slog.Error("failed to ping pool", "error", err)
		return nil, fmt.Errorf("failed to ping pool: %w", err)
	}

	slog.Info("postgres connector created successfully")
	return &Connector{
		pool: pool,
		cfg:  cfg,
	}, nil
}

func (c *Connector) Close() error {
	slog.Info("closing postgres connector")
	c.pool.Close()
	return nil
}

func (c *Connector) Pool() *pgxpool.Pool {
	slog.Info("getting postgres pool")
	return c.pool
}

func (c *Connector) HealthCheck(ctx context.Context) error {
	slog.Info("checking postgres health")
	return c.pool.Ping(ctx)
}

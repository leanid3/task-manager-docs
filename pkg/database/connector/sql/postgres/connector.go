package postgres

import (
	"app/pkg/logger"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Connector struct {
	pool *pgxpool.Pool
	cfg  *Config
	l    logger.Interface
}

func NewConnector(cfg *Config, l logger.Interface) (*Connector, error) {
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
		l.Error("failed to parse connection string", "error", err, "host", cfg.Host, "port", cfg.Port, "database", cfg.Database)
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Настраиваем пул
	poolConfig.MaxConns = cfg.MaxConnections
	poolConfig.MinConns = cfg.MinConnections
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		l.Error("failed to create connection pool", "error", err, "host", cfg.Host, "port", cfg.Port, "database", cfg.Database)
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Проверяем, что соединение работает
	if err := pool.Ping(context.Background()); err != nil {
		l.Error("failed to ping database", "error", err, "host", cfg.Host, "port", cfg.Port, "database", cfg.Database)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	l.Info("postgres connector created",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.Database,
		"max_connections", cfg.MaxConnections,
		"min_connections", cfg.MinConnections)
	return &Connector{
		pool: pool,
		cfg:  cfg,
		l:    l,
	}, nil
}

func (c *Connector) Close() error {
	c.l.Info("closing postgres connector")
	c.pool.Close()
	return nil
}

func (c *Connector) Pool() *pgxpool.Pool {
	return c.pool
}

func (c *Connector) HealthCheck(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

// TODO доделать методы для работы с БД
func (c *Connector) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return c.pool.QueryRow(ctx, query, args...)
}

func (c *Connector) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	return c.pool.Exec(ctx, query, args...)
}

// Utility для маппинга generic результатов
func (c *Connector) ScanRow(row pgx.Row, dest ...any) error {
	return row.Scan(dest...)
}

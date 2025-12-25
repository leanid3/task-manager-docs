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
	l.Info("success - parse connection string", "connString", connString)
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		l.Error("failed - parse connection string", "error", err)
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Настраиваем пул
	poolConfig.MaxConns = cfg.MaxConnections
	poolConfig.MinConns = cfg.MinConnections
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		l.Error("failed - create pool", "error", err)
		return nil, fmt.Errorf("failed - create pool: %w", err)
	}

	// Проверяем, что соединение работает
	if err := pool.Ping(context.Background()); err != nil {
		l.Error("failed - ping pool", "error", err)
		return nil, fmt.Errorf("failed - ping pool: %w", err)
	}

	l.Info("success - postgres connector created")
	return &Connector{
		pool: pool,
		cfg:  cfg,
		l:    l,
	}, nil
}

func (c *Connector) Close() error {
	c.l.Info("success - closing postgres connector")
	c.pool.Close()
	return nil
}

func (c *Connector) Pool() *pgxpool.Pool {
	c.l.Info("success - getting postgres pool")
	return c.pool
}

func (c *Connector) HealthCheck(ctx context.Context) error {
	c.l.Info("success - checking postgres health")
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

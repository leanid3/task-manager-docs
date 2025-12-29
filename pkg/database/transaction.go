package database

import (
	"app/pkg/logger"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TODO нужен рефактринг
type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

var (
	_ DB = (*pgxpool.Pool)(nil)
	_ DB = (*pgx.Conn)(nil)
	_ DB = (pgx.Tx)(nil)
)

func WithTransaction(ctx context.Context, pool *pgxpool.Pool, l logger.Interface, fn func(tx pgx.Tx) error) error {
	l.Debug("beginning transaction")
	tx, err := pool.Begin(ctx)
	if err != nil {
		l.Error("failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	var fnErr error
	defer func() {
		if p := recover(); p != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				l.Error("failed to rollback transaction after panic", "error", rollbackErr, "panic", p)
			}
			panic(p)
		} else if fnErr != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				l.Error("failed to rollback transaction", "error", rollbackErr)
			}
		} else {
			if commitErr := tx.Commit(ctx); commitErr != nil {
				l.Error("failed to commit transaction", "error", commitErr)
				fnErr = fmt.Errorf("failed to commit transaction: %w", commitErr)
			} else {
				l.Debug("transaction committed successfully")
			}
		}
	}()
	fnErr = fn(tx)
	if fnErr != nil {
		l.Error("transaction failed", "error", fnErr)
		return fmt.Errorf("transaction failed: %w", fnErr)
	}
	return fnErr
}

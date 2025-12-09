package database

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgx.CommandTag, error)
}

var (
	_ DB = (*pgxpool.Pool)(nil)
	_ DB = (*pgx.Conn)(nil)
)

func WithTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	slog.Info("beginning transaction")
	tx, err := pool.Begin(ctx)
	if err != nil {
		slog.Error("failed to begin transaction", "error", err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			if err := tx.Rollback(ctx); err != nil {
				slog.Error("failed to rollback transaction", "error", err)
			}
			panic(p)
		} else if err != nil {
			if err := tx.Rollback(ctx); err != nil {
				slog.Error("failed to rollback transaction", "error", err)
			}
		} else {
			err = tx.Commit(ctx)
		}
	}()
	err = fn(tx)
	if err != nil {
		slog.Error("transaction failed", "error", err)
		return fmt.Errorf("transaction failed: %w", err)
	}
	return nil
}

package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type KeepAlive struct {
	pool *pgxpool.Pool
}

func NewKeepAlive(pool *pgxpool.Pool) *KeepAlive {
	return &KeepAlive{pool: pool}
}

func (r *KeepAlive) Beat(ctx context.Context) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("keepalive begin: %w", err)
	}
	defer tx.Rollback(ctx)

	payload := fmt.Sprintf("%d", time.Now().UnixNano())
	var id int
	if err := tx.QueryRow(ctx, `
		INSERT INTO db_keep_alive (payload, "createdAt") VALUES ($1, now()) RETURNING id`, payload).
		Scan(&id); err != nil {
		return fmt.Errorf("keepalive insert: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE db_keep_alive SET payload = $2 WHERE id = $1`, id, payload+"-u"); err != nil {
		return fmt.Errorf("keepalive update: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM db_keep_alive WHERE "createdAt" < now() - interval '24 hours'`); err != nil {
		return fmt.Errorf("keepalive delete: %w", err)
	}
	var n int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM db_keep_alive`).Scan(&n); err != nil {
		return fmt.Errorf("keepalive count: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("keepalive commit: %w", err)
	}
	return nil
}

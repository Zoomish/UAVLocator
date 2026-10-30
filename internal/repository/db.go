package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgx pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgx ping: %w", err)
	}
	return pool, nil
}

func Probe(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		SELECT "tgId", admin, notifications, name, username, locations, "createdAt", "updatedAt"
		FROM "user" WHERE false`)
	if err != nil {
		return fmt.Errorf("user table is missing columns from the current bot schema: %w", err)
	}
	_, err = pool.Exec(ctx, `
		SELECT id, msg_id, "waitingFor", msg_id_delete, "tgId"
		FROM bot WHERE false`)
	if err != nil {
		return fmt.Errorf("bot table is missing columns from the current bot schema: %w", err)
	}
	return nil
}

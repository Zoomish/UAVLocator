package migrate

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed sql/*.sql
var files embed.FS

func Up(ctx context.Context, pool *pgxpool.Pool) error {
	goose.SetBaseFS(files)
	goose.SetTableName("locator_schema_migrations")
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	if err := goose.UpContext(ctx, db, "sql"); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

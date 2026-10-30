package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"locatorbpla/internal/domain"
)

const userCols = `id, "tgId", admin, notifications, name, username, COALESCE(locations, '{}')`

type Users struct {
	pool *pgxpool.Pool
}

func NewUsers(pool *pgxpool.Pool) *Users {
	return &Users{pool: pool}
}

func (r *Users) Upsert(ctx context.Context, tgID int64, name, username string) (domain.User, error) {
	if name == "" {
		name = "—"
	}
	var uname *string
	if username != "" {
		uname = &username
	}
	row := r.pool.QueryRow(ctx, `
		INSERT INTO "user" ("tgId", admin, notifications, name, username, locations, "createdAt", "updatedAt")
		VALUES ($1, false, true, $2, $3, '{}', now(), now())
		ON CONFLICT ("tgId") DO UPDATE
		SET name = EXCLUDED.name, username = EXCLUDED.username, "updatedAt" = now()
		RETURNING `+userCols, tgID, name, uname)
	user, err := scanUser(row)
	if err != nil {
		return domain.User{}, fmt.Errorf("upsert user: %w", err)
	}
	return user, nil
}

func (r *Users) Find(ctx context.Context, tgID int64) (domain.User, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+userCols+` FROM "user" WHERE "tgId" = $1`, tgID)
	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, err
		}
		return domain.User{}, fmt.Errorf("find user: %w", err)
	}
	return user, nil
}

func (r *Users) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+userCols+` FROM "user" ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		out = append(out, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return out, nil
}

func (r *Users) FindNotifiable(ctx context.Context, normalized string) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+userCols+`
		FROM "user"
		WHERE notifications = true
		  AND EXISTS (
		    SELECT 1 FROM unnest(locations) AS pattern_item
		    WHERE pattern_item <> '' AND position(pattern_item in $1) > 0
		  )`, normalized)
	if err != nil {
		return nil, fmt.Errorf("find notifiable: %w", err)
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan notifiable: %w", err)
		}
		out = append(out, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find notifiable: %w", err)
	}
	return out, nil
}

func (r *Users) SetNotifications(ctx context.Context, tgID int64, on bool) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE "user" SET notifications = $2, "updatedAt" = now() WHERE "tgId" = $1`, tgID, on)
	if err != nil {
		return fmt.Errorf("set notifications: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Users) SetLocations(ctx context.Context, tgID int64, locations []string) error {
	if locations == nil {
		locations = []string{}
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE "user" SET locations = $2, "updatedAt" = now() WHERE "tgId" = $1`, tgID, locations)
	if err != nil {
		return fmt.Errorf("set locations: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Users) ToggleAdmin(ctx context.Context, tgID int64) (domain.User, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE "user" SET admin = NOT admin, "updatedAt" = now() WHERE "tgId" = $1
		RETURNING `+userCols, tgID)
	user, err := scanUser(row)
	if err != nil {
		return domain.User{}, fmt.Errorf("toggle admin: %w", err)
	}
	return user, nil
}

func scanUser(row pgx.Row) (domain.User, error) {
	var user domain.User
	var username *string
	err := row.Scan(&user.ID, &user.TgID, &user.Admin, &user.Notifications, &user.Name, &username, &user.Locations)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	if username != nil {
		user.Username = *username
	}
	if user.Locations == nil {
		user.Locations = []string{}
	}
	return user, nil
}

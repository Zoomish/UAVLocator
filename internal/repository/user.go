package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"locatorbpla/internal/domain"
)

const userCols = `id, "tgId", admin, notifications, name, username, COALESCE(locations, '{}'), COALESCE("locationsEditorId", 0), COALESCE("locationsEditor", ''), COALESCE(webhook_url, '')`

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

func (r *Users) FindNotifiable(ctx context.Context, normalized string) ([]domain.Match, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+userCols+`, (
			SELECT pattern_item
			FROM unnest(locations) AS pattern_item
			WHERE pattern_item <> '' AND position(pattern_item in $1) > 0
			ORDER BY char_length(pattern_item) DESC, pattern_item
			LIMIT 1
		)
		FROM "user"
		WHERE notifications = true
		  AND blocked = false
		  AND EXISTS (
		    SELECT 1 FROM unnest(locations) AS pattern_item
		    WHERE pattern_item <> '' AND position(pattern_item in $1) > 0
		  )`, normalized)
	if err != nil {
		return nil, fmt.Errorf("find notifiable: %w", err)
	}
	defer rows.Close()
	var out []domain.Match
	for rows.Next() {
		user, pattern, err := scanMatch(rows)
		if err != nil {
			return nil, fmt.Errorf("scan notifiable: %w", err)
		}
		if pattern == "" {
			continue
		}
		out = append(out, domain.Match{User: user, Pattern: pattern})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("find notifiable: %w", err)
	}
	return out, nil
}

func (r *Users) SetNotifications(ctx context.Context, tgID int64, on bool) error {
	query := `UPDATE "user" SET notifications = false, "updatedAt" = now() WHERE "tgId" = $1`
	if on {
		query = `UPDATE "user" SET notifications = true, blocked = false, "updatedAt" = now() WHERE "tgId" = $1`
	}
	tag, err := r.pool.Exec(ctx, query, tgID)
	if err != nil {
		return fmt.Errorf("set notifications: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Users) ReleaseBlock(ctx context.Context, tgID int64) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE "user" SET blocked = false, notifications = false, "updatedAt" = now()
		WHERE "tgId" = $1 AND blocked = true`, tgID)
	if err != nil {
		return false, fmt.Errorf("release block: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *Users) Block(ctx context.Context, tgID int64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE "user" SET notifications = false, blocked = true, "updatedAt" = now() WHERE "tgId" = $1`, tgID)
	if err != nil {
		return fmt.Errorf("block user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Users) Audience(ctx context.Context) (domain.Audience, error) {
	var stats domain.Audience
	err := r.pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE cardinality(COALESCE(locations, '{}')) = 0),
			count(*) FILTER (WHERE notifications = false AND blocked = false),
			count(*) FILTER (WHERE blocked)
		FROM "user"`).Scan(&stats.NoLocations, &stats.Muted, &stats.Blocked)
	if err != nil {
		return domain.Audience{}, fmt.Errorf("audience: %w", err)
	}
	return stats, nil
}

func (r *Users) SetLocations(ctx context.Context, tgID int64, locations []string, editorID int64, editor string) error {
	if locations == nil {
		locations = []string{}
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE "user" SET locations = $2,
			"locationsEditorId" = CASE WHEN $3 = 0 THEN "locationsEditorId" ELSE $3 END,
			"locationsEditor" = CASE WHEN $4 = '' THEN "locationsEditor" ELSE $4 END,
			"updatedAt" = now()
		WHERE "tgId" = $1`, tgID, locations, editorID, editor)
	if err != nil {
		return fmt.Errorf("set locations: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Users) SetWebhook(ctx context.Context, tgID int64, raw string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE "user" SET webhook_url = NULLIF($2, ''), "updatedAt" = now() WHERE "tgId" = $1`, tgID, raw)
	if err != nil {
		return fmt.Errorf("set webhook: %w", err)
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
	err := row.Scan(&user.ID, &user.TgID, &user.Admin, &user.Notifications, &user.Name, &username, &user.Locations, &user.LocationsEditorID, &user.LocationsEditor, &user.WebhookURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	return finishUser(user, username), nil
}

func scanMatch(row pgx.Row) (domain.User, string, error) {
	var user domain.User
	var username *string
	var pattern *string
	err := row.Scan(&user.ID, &user.TgID, &user.Admin, &user.Notifications, &user.Name, &username, &user.Locations, &user.LocationsEditorID, &user.LocationsEditor, &user.WebhookURL, &pattern)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, "", domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, "", err
	}
	matched := ""
	if pattern != nil {
		matched = *pattern
	}
	return finishUser(user, username), matched, nil
}

func finishUser(user domain.User, username *string) domain.User {
	if username != nil {
		user.Username = *username
	}
	if user.Locations == nil {
		user.Locations = []string{}
	}
	return user
}

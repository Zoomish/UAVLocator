package repository

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"locatorbpla/internal/domain"
)

type MTSession struct {
	pool *pgxpool.Pool
}

func NewMTSession(pool *pgxpool.Pool) *MTSession {
	return &MTSession{pool: pool}
}

func DecodeSession(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	buf, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		buf, err = base64.RawStdEncoding.DecodeString(raw)
	}
	if err != nil {
		return nil, fmt.Errorf("decode session: %w", err)
	}
	return buf, nil
}

func EncodeSession(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func SessionSeed(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (r *MTSession) Load(ctx context.Context) (data []byte, seed string, err error) {
	err = r.pool.QueryRow(ctx, `SELECT data, seed FROM tg_session WHERE id = 1`).Scan(&data, &seed)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, "", domain.ErrNotFound
	}
	if err != nil {
		return nil, "", fmt.Errorf("load mt session: %w", err)
	}
	return data, seed, nil
}

func (r *MTSession) Save(ctx context.Context, data []byte, seed string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO tg_session (id, data, seed, "updatedAt")
		VALUES (1, $1, $2, now())
		ON CONFLICT (id) DO UPDATE
		SET data = EXCLUDED.data,
		    seed = CASE WHEN EXCLUDED.seed = '' THEN tg_session.seed ELSE EXCLUDED.seed END,
		    "updatedAt" = now()`, data, seed)
	if err != nil {
		return fmt.Errorf("save mt session: %w", err)
	}
	return nil
}

func (r *MTSession) Bootstrap(ctx context.Context, raw string) ([]byte, error) {
	envBlob, err := DecodeSession(raw)
	if err != nil {
		return nil, fmt.Errorf("TG_SESSION: %w", err)
	}
	dbBlob, dbSeed, err := r.Load(ctx)
	if errors.Is(err, domain.ErrNotFound) {
		dbBlob, dbSeed, err = nil, "", nil
	}
	if err != nil {
		return nil, err
	}
	data, seed, write := chooseSession(envBlob, dbBlob, dbSeed)
	if write {
		if err := r.Save(ctx, data, seed); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func chooseSession(envBlob, dbBlob []byte, dbSeed string) (data []byte, seed string, write bool) {
	envSeed := SessionSeed(envBlob)
	switch {
	case len(envBlob) > 0 && (len(dbBlob) == 0 || dbSeed != envSeed):
		return envBlob, envSeed, true
	case len(dbBlob) > 0:
		return dbBlob, dbSeed, false
	default:
		return nil, "", false
	}
}

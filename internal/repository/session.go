package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"locatorbpla/internal/domain"
)

type Sessions struct {
	pool *pgxpool.Pool
}

func NewSessions(pool *pgxpool.Pool) *Sessions {
	return &Sessions{pool: pool}
}

func (r *Sessions) Ensure(ctx context.Context, tgID int64) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO bot ("tgId") VALUES ($1) ON CONFLICT ("tgId") DO NOTHING`, tgID)
	if err != nil {
		return fmt.Errorf("ensure session: %w", err)
	}
	return nil
}

func (r *Sessions) Load(ctx context.Context, tgID int64) (domain.Session, error) {
	var waiting, waitingName, draft sql.NullString
	var msgID, errID, waitingUser, draftMsg sql.NullInt64
	err := r.pool.QueryRow(ctx, `
		SELECT "waitingFor", msg_id, msg_id_delete, "waitingUser", "waitingName", draft, "draftMsg"
		FROM bot WHERE "tgId" = $1`, tgID).
		Scan(&waiting, &msgID, &errID, &waitingUser, &waitingName, &draft, &draftMsg)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Session{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Session{}, fmt.Errorf("load session: %w", err)
	}
	s := domain.Session{TgID: tgID}
	if waiting.Valid {
		s.WaitingFor = waiting.String
	}
	if msgID.Valid {
		s.MsgID = int(msgID.Int64)
	}
	if errID.Valid {
		s.ErrorMsgID = int(errID.Int64)
	}
	if waitingUser.Valid {
		s.WaitingUser = waitingUser.Int64
	}
	if waitingName.Valid {
		s.WaitingName = waitingName.String
	}
	if draft.Valid {
		s.Draft = draft.String
	}
	if draftMsg.Valid {
		s.DraftMsg = int(draftMsg.Int64)
	}
	return s, nil
}

func nullMsgID(msgID int) any {
	if msgID == 0 {
		return nil
	}
	return msgID
}

func (r *Sessions) SetFunctional(ctx context.Context, tgID int64, msgID int) error {
	tag, err := r.pool.Exec(ctx, `UPDATE bot SET msg_id = $2 WHERE "tgId" = $1`, tgID, nullMsgID(msgID))
	if err != nil {
		return fmt.Errorf("set functional message: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Sessions) SetError(ctx context.Context, tgID int64, msgID int) error {
	tag, err := r.pool.Exec(ctx, `UPDATE bot SET msg_id_delete = $2 WHERE "tgId" = $1`, tgID, nullMsgID(msgID))
	if err != nil {
		return fmt.Errorf("set error message: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Sessions) SetWaiting(ctx context.Context, tgID int64, waiting string, msgID int, actor int64, name string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE bot SET "waitingFor" = $2, msg_id = COALESCE($3, msg_id),
			"waitingUser" = NULLIF($4::bigint, 0),
			"waitingName" = CASE WHEN $4::bigint = 0 THEN NULL ELSE NULLIF($5, '') END
		WHERE "tgId" = $1`, tgID, waiting, nullMsgID(msgID), actor, name)
	if err != nil {
		return fmt.Errorf("set waiting: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Sessions) SetDraft(ctx context.Context, tgID int64, draft string, msgID int) error {
	var tag interface{ RowsAffected() int64 }
	var err error
	if msgID == 0 {
		tag, err = r.pool.Exec(ctx, `UPDATE bot SET draft = NULLIF($2, '') WHERE "tgId" = $1`, tgID, draft)
	} else {
		tag, err = r.pool.Exec(ctx, `UPDATE bot SET draft = NULLIF($2, ''), "draftMsg" = $3 WHERE "tgId" = $1`, tgID, draft, msgID)
	}
	if err != nil {
		return fmt.Errorf("set draft: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Sessions) ClearWaiting(ctx context.Context, tgID int64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE bot SET "waitingFor" = NULL, "waitingUser" = NULL, "waitingName" = NULL, draft = NULL, "draftMsg" = NULL
		WHERE "tgId" = $1`, tgID)
	if err != nil {
		return fmt.Errorf("clear waiting: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Sessions) ResetAllWaiting(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE bot SET "waitingFor" = NULL, "waitingUser" = NULL, "waitingName" = NULL, draft = NULL, "draftMsg" = NULL, msg_id = NULL, msg_id_delete = NULL`)
	if err != nil {
		return 0, fmt.Errorf("reset waiting: %w", err)
	}
	return tag.RowsAffected(), nil
}

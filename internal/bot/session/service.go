package session

import (
	"context"

	"locatorbpla/internal/domain"
)

type Repo interface {
	Load(ctx context.Context, tgID int64) (domain.Session, error)
	Ensure(ctx context.Context, tgID int64) error
	SetFunctional(ctx context.Context, tgID int64, msgID int) error
	SetError(ctx context.Context, tgID int64, msgID int) error
	SetWaiting(ctx context.Context, tgID int64, waiting string, msgID int, actor int64, name string) error
	SetDraft(ctx context.Context, tgID int64, draft string, msgID int) error
	ClearWaiting(ctx context.Context, tgID int64) error
}

type Service struct {
	Repo Repo
}

func (s *Service) Load(ctx context.Context, tgID int64) (domain.Session, error) {
	return s.Repo.Load(ctx, tgID)
}

func (s *Service) Ensure(ctx context.Context, tgID int64) error {
	return s.Repo.Ensure(ctx, tgID)
}

func (s *Service) SetFunctional(ctx context.Context, tgID int64, msgID int) error {
	return s.Repo.SetFunctional(ctx, tgID, msgID)
}

func (s *Service) SetError(ctx context.Context, tgID int64, msgID int) error {
	return s.Repo.SetError(ctx, tgID, msgID)
}

func (s *Service) SetWaiting(ctx context.Context, tgID int64, waiting string, msgID int, actor int64, name string) error {
	return s.Repo.SetWaiting(ctx, tgID, waiting, msgID, actor, name)
}

func (s *Service) SetDraft(ctx context.Context, tgID int64, draft string, msgID int) error {
	return s.Repo.SetDraft(ctx, tgID, draft, msgID)
}

func (s *Service) ClearWaiting(ctx context.Context, tgID int64) error {
	return s.Repo.ClearWaiting(ctx, tgID)
}

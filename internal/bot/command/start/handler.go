package start

import (
	"context"
	"fmt"

	"locatorbpla/internal/bot/screen"
	"locatorbpla/internal/bot/session"
	"locatorbpla/internal/bot/view"
	"locatorbpla/internal/repository"
)

type Handler struct {
	Users    *repository.Users
	Sessions *session.Service
	Screen   *screen.Screen
}

func (h *Handler) Command(ctx context.Context, chatID int64, name, username string, userMsgID int, group bool) error {
	user, err := h.Users.Upsert(ctx, chatID, name, username)
	if err != nil {
		return fmt.Errorf("start upsert: %w", err)
	}
	if _, err := h.Users.ReleaseBlock(ctx, chatID); err != nil {
		return fmt.Errorf("start release: %w", err)
	}
	if err := h.Sessions.Ensure(ctx, chatID); err != nil {
		return fmt.Errorf("start session: %w", err)
	}
	text, kb := view.Welcome()
	if group {
		text, kb = view.GroupBoard(user)
	}
	if err := h.Screen.SendCommand(ctx, chatID, text, kb); err != nil {
		return fmt.Errorf("start screen: %w", err)
	}
	if err := h.Screen.ClearNoise(ctx, chatID, userMsgID); err != nil {
		return fmt.Errorf("start cleanup: %w", err)
	}
	return nil
}

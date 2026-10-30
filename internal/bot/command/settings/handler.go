package settings

import (
	"context"
	"errors"
	"fmt"

	"locatorbpla/internal/bot/screen"
	"locatorbpla/internal/bot/session"
	"locatorbpla/internal/bot/view"
	"locatorbpla/internal/domain"
	"locatorbpla/internal/repository"
)

type Handler struct {
	Users    *repository.Users
	Sessions *session.Service
	Screen   *screen.Screen
}

func (h *Handler) Command(ctx context.Context, chatID int64, userMsgID int) error {
	if err := h.Sessions.Ensure(ctx, chatID); err != nil {
		return fmt.Errorf("settings session: %w", err)
	}
	user, err := h.Users.Find(ctx, chatID)
	if errors.Is(err, domain.ErrNotFound) {
		h.Screen.Delete(chatID, userMsgID)
		return h.Screen.SendUserError(ctx, chatID, "Сначала отправьте /start.")
	}
	if err != nil {
		return fmt.Errorf("settings user: %w", err)
	}
	text, kb := view.Settings(user)
	if err := h.Screen.SendCommand(ctx, chatID, text, kb); err != nil {
		return fmt.Errorf("settings screen: %w", err)
	}
	if err := h.Screen.ClearNoise(ctx, chatID, userMsgID); err != nil {
		return fmt.Errorf("settings cleanup: %w", err)
	}
	return nil
}

func (h *Handler) Edit(ctx context.Context, chatID int64, messageID int) error {
	user, err := h.Users.Find(ctx, chatID)
	if err != nil {
		return fmt.Errorf("settings user: %w", err)
	}
	text, kb := view.Settings(user)
	if err := h.Screen.EditFunctional(ctx, chatID, messageID, text, kb); err != nil {
		return fmt.Errorf("settings edit: %w", err)
	}
	return nil
}

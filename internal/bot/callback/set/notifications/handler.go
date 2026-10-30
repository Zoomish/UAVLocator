package notifications

import (
	"context"
	"fmt"

	"locatorbpla/internal/bot/cb"
	"locatorbpla/internal/bot/command/settings"
	"locatorbpla/internal/bot/screen"
	"locatorbpla/internal/repository"
)

type Handler struct {
	Users    *repository.Users
	Settings *settings.Handler
	Screen   *screen.Screen
}

func (h *Handler) Handle(ctx context.Context, q cb.Query, on bool) error {
	verb := "выключаете"
	if on {
		verb = "включаете"
	}
	h.Screen.Toast(q.ID, "Вы "+verb+" уведомления")
	if err := h.Users.SetNotifications(ctx, q.ChatID, on); err != nil {
		return fmt.Errorf("set notifications: %w", err)
	}
	if err := h.Settings.Edit(ctx, q.ChatID, q.MessageID); err != nil {
		return fmt.Errorf("refresh settings: %w", err)
	}
	return nil
}

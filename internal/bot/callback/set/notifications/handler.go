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
	if q.Group {
		ok, err := h.Screen.IsChatAdmin(ctx, q.ChatID, q.Actor)
		if err != nil || !ok {
			h.Screen.Toast(ctx, q.ID, "Недостаточно прав")
			return nil
		}
	}
	verb := "выключаете"
	if on {
		verb = "включаете"
	}
	if err := h.Users.SetNotifications(ctx, q.ChatID, on); err != nil {
		return fmt.Errorf("set notifications: %w", err)
	}
	public, err := h.Settings.Show(ctx, q)
	if err != nil {
		return fmt.Errorf("refresh settings: %w", err)
	}
	if !q.Group || public {
		h.Screen.Toast(ctx, q.ID, "Вы "+verb+" уведомления")
	}
	return nil
}

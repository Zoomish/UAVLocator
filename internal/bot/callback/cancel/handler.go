package cancel

import (
	"context"
	"fmt"

	"locatorbpla/internal/bot/cb"
	"locatorbpla/internal/bot/command/settings"
	"locatorbpla/internal/bot/screen"
	"locatorbpla/internal/bot/session"
)

type Handler struct {
	Sessions *session.Service
	Settings *settings.Handler
	Screen   *screen.Screen
}

func (h *Handler) Handle(ctx context.Context, q cb.Query, rest string) error {
	head, _ := cb.Split(rest)
	switch head {
	case "locations":
		h.Screen.Toast(q.ID, "Вы отменили отправку локаций")
		if err := h.Sessions.ClearWaiting(ctx, q.ChatID); err != nil {
			return fmt.Errorf("cancel locations: %w", err)
		}
		if err := h.Screen.ClearNoise(ctx, q.ChatID, 0); err != nil {
			return fmt.Errorf("cancel locations cleanup: %w", err)
		}
		return h.Settings.Edit(ctx, q.ChatID, q.MessageID)
	case "error":
		h.Screen.Toast(q.ID, "Вы отменили отправку ошибки")
		if err := h.Sessions.ClearWaiting(ctx, q.ChatID); err != nil {
			return fmt.Errorf("cancel error: %w", err)
		}
		if err := h.Screen.ClearNoise(ctx, q.ChatID, 0); err != nil {
			return fmt.Errorf("cancel error cleanup: %w", err)
		}
		h.Screen.Delete(q.ChatID, q.MessageID)
		if err := h.Sessions.SetFunctional(ctx, q.ChatID, 0); err != nil {
			return fmt.Errorf("clear cancelled message: %w", err)
		}
		return nil
	default:
		h.Screen.Toast(q.ID, "Неизвестная отмена")
		return nil
	}
}

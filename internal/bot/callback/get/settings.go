package get

import (
	"context"
	"fmt"

	"locatorbpla/internal/bot/cb"
	"locatorbpla/internal/bot/command/settings"
	"locatorbpla/internal/bot/screen"
)

type Settings struct {
	Settings *settings.Handler
	Screen   *screen.Screen
}

func (h *Settings) Handle(ctx context.Context, q cb.Query) error {
	h.Screen.Toast(q.ID, "Перейти в настройки")
	if err := h.Settings.Edit(ctx, q.ChatID, q.MessageID); err != nil {
		return fmt.Errorf("get settings: %w", err)
	}
	return nil
}

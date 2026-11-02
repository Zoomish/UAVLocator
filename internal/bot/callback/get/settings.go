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
	public, err := h.Settings.Show(ctx, q)
	if err != nil {
		return fmt.Errorf("get settings: %w", err)
	}
	if !q.Group || public {
		h.Screen.Toast(ctx, q.ID, "Перейти в настройки")
	}
	return nil
}

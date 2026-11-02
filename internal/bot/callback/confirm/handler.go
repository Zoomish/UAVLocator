package confirm

import (
	"context"

	"locatorbpla/internal/bot/cb"
	"locatorbpla/internal/bot/input"
	"locatorbpla/internal/bot/screen"
)

type Handler struct {
	Input  *input.Router
	Screen *screen.Screen
}

func (h *Handler) Handle(ctx context.Context, q cb.Query, action string) error {
	if h.Input == nil {
		h.Screen.Toast(ctx, q.ID, "Недоступно")
		return nil
	}
	return h.Input.Confirm(ctx, q, action)
}

package locations

import (
	"context"
	"fmt"

	"locatorbpla/internal/bot/cb"
	"locatorbpla/internal/bot/screen"
	"locatorbpla/internal/bot/session"
	"locatorbpla/internal/bot/view"
	"locatorbpla/internal/domain"
)

type Handler struct {
	Sessions *session.Service
	Screen   *screen.Screen
}

func (h *Handler) Handle(ctx context.Context, q cb.Query) error {
	h.Screen.Toast(q.ID, "Перечислите локации")
	if err := h.Sessions.SetWaiting(ctx, q.ChatID, domain.WaitLocations, q.MessageID); err != nil {
		return fmt.Errorf("wait locations: %w", err)
	}
	text, kb := view.LocationsPrompt()
	if err := h.Screen.EditFunctional(ctx, q.ChatID, q.MessageID, text, kb); err != nil {
		_ = h.Sessions.ClearWaiting(ctx, q.ChatID)
		return fmt.Errorf("locations prompt: %w", err)
	}
	return nil
}

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
	if q.Group {
		ok, err := h.Screen.IsChatAdmin(ctx, q.ChatID, q.Actor)
		if err != nil || !ok {
			h.Screen.Toast(ctx, q.ID, "Недостаточно прав")
			return nil
		}
	}
	msgID := q.MessageID
	actor := int64(0)
	if q.Group {
		msgID = 0
		actor = q.Actor
	}
	name := ""
	if q.Group {
		name = q.Name
	}
	if err := h.Sessions.SetWaiting(ctx, q.ChatID, domain.WaitLocations, msgID, actor, name); err != nil {
		return fmt.Errorf("wait locations: %w", err)
	}
	text, kb := view.LocationsPrompt()
	if q.Group {
		hidden := view.HiddenIntro(q.Name, true) + text
		_, _, err := h.Screen.Whisper(ctx, q.ChatID, screen.Talk{
			Actor: q.Actor, CallbackID: q.ID, EphemeralID: q.EphemeralID, Replace: q.EphemeralID == 0,
		}, hidden, &kb)
		if err != nil {
			if _, plainErr := h.Screen.SendPlain(ctx, q.ChatID, text, &kb); plainErr != nil {
				_ = h.Sessions.ClearWaiting(ctx, q.ChatID)
				return fmt.Errorf("locations prompt: %w", err)
			}
			h.Screen.Toast(ctx, q.ID, "Перечислите локации")
		}
		return nil
	}
	if err := h.Screen.EditFunctional(ctx, q.ChatID, q.MessageID, text, kb); err != nil {
		_ = h.Sessions.ClearWaiting(ctx, q.ChatID)
		return fmt.Errorf("locations prompt: %w", err)
	}
	h.Screen.Toast(ctx, q.ID, "Перечислите локации")
	return nil
}

package hook

import (
	"context"
	"fmt"

	"locatorbpla/internal/bot/cb"
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

func (h *Handler) Open(ctx context.Context, q cb.Query) error {
	if q.Group {
		h.Screen.Toast(ctx, q.ID, "Недостаточно прав")
		return nil
	}
	user, err := h.Users.Find(ctx, q.ChatID)
	if err != nil {
		return fmt.Errorf("api user: %w", err)
	}
	text, kb := view.APIMenu(user)
	if err := h.Screen.EditFunctional(ctx, q.ChatID, q.MessageID, text, kb); err != nil {
		return fmt.Errorf("api menu: %w", err)
	}
	h.Screen.Toast(ctx, q.ID, "API")
	return nil
}

func (h *Handler) Change(ctx context.Context, q cb.Query) error {
	if q.Group {
		h.Screen.Toast(ctx, q.ID, "Недостаточно прав")
		return nil
	}
	if err := h.Sessions.SetWaiting(ctx, q.ChatID, domain.WaitWebhook, q.MessageID, 0, ""); err != nil {
		return fmt.Errorf("wait api: %w", err)
	}
	text, kb := view.APIPrompt(false)
	if err := h.Screen.EditFunctional(ctx, q.ChatID, q.MessageID, text, kb); err != nil {
		_ = h.Sessions.ClearWaiting(ctx, q.ChatID)
		return fmt.Errorf("api prompt: %w", err)
	}
	h.Screen.Toast(ctx, q.ID, "Пришлите URL")
	return nil
}

func (h *Handler) Disable(ctx context.Context, q cb.Query) error {
	if q.Group {
		h.Screen.Toast(ctx, q.ID, "Недостаточно прав")
		return nil
	}
	user, err := h.Users.Find(ctx, q.ChatID)
	if err != nil {
		return fmt.Errorf("api user: %w", err)
	}
	if user.WebhookURL == "" {
		text, kb := view.APIMenu(user)
		if err := h.Screen.EditFunctional(ctx, q.ChatID, q.MessageID, text, kb); err != nil {
			return fmt.Errorf("api menu: %w", err)
		}
		h.Screen.Toast(ctx, q.ID, "Адрес и так не задан")
		return nil
	}
	if err := h.Users.SetWebhook(ctx, q.ChatID, ""); err != nil {
		return fmt.Errorf("clear api: %w", err)
	}
	user.WebhookURL = ""
	text, kb := view.APIMenu(user)
	if err := h.Screen.EditFunctional(ctx, q.ChatID, q.MessageID, text, kb); err != nil {
		return fmt.Errorf("api menu: %w", err)
	}
	h.Screen.Toast(ctx, q.ID, "API отключён")
	return nil
}

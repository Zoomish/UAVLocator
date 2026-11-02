package cancel

import (
	"context"
	"fmt"

	"errors"

	"locatorbpla/internal/bot/cb"
	"locatorbpla/internal/bot/command/settings"
	"locatorbpla/internal/bot/screen"
	"locatorbpla/internal/bot/session"
	"locatorbpla/internal/bot/view"
	"locatorbpla/internal/domain"
	"locatorbpla/internal/repository"
)

type Handler struct {
	Users    *repository.Users
	Sessions *session.Service
	Settings *settings.Handler
	Screen   *screen.Screen
}

func cancelMatches(head, waiting string) bool {
	switch head {
	case "locations":
		return waiting == domain.WaitLocations
	case "api", "webhook":
		return waiting == domain.WaitWebhook
	case "error":
		return waiting == domain.WaitError
	default:
		return false
	}
}

func (h *Handler) Handle(ctx context.Context, q cb.Query, rest string) error {
	head, _ := cb.Split(rest)
	sess, err := h.Sessions.Load(ctx, q.ChatID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("cancel session: %w", err)
	}
	if !cancelMatches(head, sess.WaitingFor) {
		h.Screen.Toast(ctx, q.ID, "Эта отмена уже не действует")
		return nil
	}
	switch head {
	case "api":
		if err := h.Sessions.ClearWaiting(ctx, q.ChatID); err != nil {
			return fmt.Errorf("cancel api: %w", err)
		}
		user, err := h.Users.Find(ctx, q.ChatID)
		if err != nil {
			return fmt.Errorf("api user: %w", err)
		}
		text, kb := view.APIMenu(user)
		if q.Group {
			_, _, err = h.Screen.Whisper(ctx, q.ChatID, screen.Talk{
				Actor: q.Actor, CallbackID: q.ID, EphemeralID: q.EphemeralID,
			}, text, &kb)
			if err != nil {
				h.Screen.Toast(ctx, q.ID, "Нажмите ещё раз")
			}
			return nil
		}
		if err := h.Screen.EditFunctional(ctx, q.ChatID, q.MessageID, text, kb); err != nil {
			return err
		}
		h.Screen.Toast(ctx, q.ID, "Вы отменили ввод")
		return nil
	case "locations", "webhook":
		if err := h.Sessions.ClearWaiting(ctx, q.ChatID); err != nil {
			return fmt.Errorf("cancel input: %w", err)
		}
		if err := h.Screen.ClearNoise(ctx, q.ChatID, 0); err != nil {
			return fmt.Errorf("cancel input cleanup: %w", err)
		}
		public, err := h.Settings.Show(ctx, q)
		if err != nil {
			return err
		}
		if !q.Group || public {
			h.Screen.Toast(ctx, q.ID, "Вы отменили ввод")
		}
		return nil
	case "error":
		if err := h.Sessions.ClearWaiting(ctx, q.ChatID); err != nil {
			return fmt.Errorf("cancel error: %w", err)
		}
		if err := h.Screen.ClearNoise(ctx, q.ChatID, 0); err != nil {
			return fmt.Errorf("cancel error cleanup: %w", err)
		}
		if q.Group && q.EphemeralID != 0 {
			h.Screen.DeleteEphemeral(ctx, q.ChatID, q.Actor, q.EphemeralID)
		}
		if !q.Group {
			h.Screen.Delete(ctx, q.ChatID, q.MessageID)
			if err := h.Sessions.SetFunctional(ctx, q.ChatID, 0); err != nil {
				return fmt.Errorf("clear cancelled message: %w", err)
			}
		}
		h.Screen.Toast(ctx, q.ID, "Вы отменили отправку ошибки")
		return nil
	default:
		h.Screen.Toast(ctx, q.ID, "Неизвестная отмена")
		return nil
	}
}

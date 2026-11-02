package settings

import (
	"context"
	"errors"
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

func (h *Handler) Command(ctx context.Context, chatID int64, userMsgID int, talk screen.Talk) error {
	if err := h.Sessions.Ensure(ctx, chatID); err != nil {
		return fmt.Errorf("settings session: %w", err)
	}
	user, err := h.Users.Find(ctx, chatID)
	if errors.Is(err, domain.ErrNotFound) {
		h.Screen.Delete(ctx, chatID, userMsgID)
		if chatID < 0 && talk.Actor != 0 {
			talk.AllowPublic = false
			_, _, _ = h.Screen.Whisper(ctx, chatID, talk, "Сначала отправьте /start.", nil)
			return nil
		}
		return h.Screen.SendUserError(ctx, chatID, "Сначала отправьте /start.")
	}
	if err != nil {
		return fmt.Errorf("settings user: %w", err)
	}
	text, kb := view.Settings(user, chatID > 0)
	if chatID < 0 && talk.Actor != 0 {
		talk.AllowPublic = false
		hidden := view.HiddenIntro(talk.Name, false) + text
		if _, _, err := h.Screen.Whisper(ctx, chatID, talk, hidden, &kb); err != nil {
			if _, plainErr := h.Screen.SendPlain(ctx, chatID, text, &kb); plainErr != nil {
				return fmt.Errorf("settings screen: %w", err)
			}
		}
		if err := h.Screen.ClearNoise(ctx, chatID, userMsgID); err != nil {
			return fmt.Errorf("settings cleanup: %w", err)
		}
		return nil
	}
	if err := h.Screen.SendCommand(ctx, chatID, text, kb); err != nil {
		return fmt.Errorf("settings screen: %w", err)
	}
	if err := h.Screen.ClearNoise(ctx, chatID, userMsgID); err != nil {
		return fmt.Errorf("settings cleanup: %w", err)
	}
	return nil
}

func (h *Handler) Show(ctx context.Context, q cb.Query) (bool, error) {
	if q.Group {
		user, err := h.Users.Find(ctx, q.ChatID)
		if err != nil {
			return false, fmt.Errorf("settings user: %w", err)
		}
		text, kb := view.Settings(user, false)
		hidden := view.HiddenIntro(q.Name, false) + text
		_, _, err = h.Screen.Whisper(ctx, q.ChatID, screen.Talk{
			Actor: q.Actor, CallbackID: q.ID, EphemeralID: q.EphemeralID, Replace: q.EphemeralID == 0,
		}, hidden, &kb)
		if err != nil {
			if _, plainErr := h.Screen.SendPlain(ctx, q.ChatID, text, &kb); plainErr != nil {
				return false, fmt.Errorf("settings whisper: %w", err)
			}
			return true, nil
		}
		return false, nil
	}
	return false, h.Edit(ctx, q.ChatID, q.MessageID)
}

func (h *Handler) Edit(ctx context.Context, chatID int64, messageID int) error {
	user, err := h.Users.Find(ctx, chatID)
	if err != nil {
		return fmt.Errorf("settings user: %w", err)
	}
	text, kb := view.Settings(user, true)
	if err := h.Screen.EditFunctional(ctx, chatID, messageID, text, kb); err != nil {
		return fmt.Errorf("settings edit: %w", err)
	}
	return nil
}

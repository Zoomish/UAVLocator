package errorreport

import (
	"context"
	"fmt"

	"locatorbpla/internal/bot/screen"
	"locatorbpla/internal/bot/session"
	"locatorbpla/internal/bot/view"
	"locatorbpla/internal/domain"
)

type Handler struct {
	Sessions *session.Service
	Screen   *screen.Screen
}

func (h *Handler) Command(ctx context.Context, chatID int64, userMsgID int, talk screen.Talk) error {
	if err := h.Sessions.Ensure(ctx, chatID); err != nil {
		return fmt.Errorf("error session: %w", err)
	}
	text, kb := view.ErrorPrompt()
	if chatID < 0 && talk.Actor != 0 {
		talk.AllowPublic = false
		text = view.HiddenIntro(talk.Name, true) + text
		id, public, err := h.Screen.Whisper(ctx, chatID, talk, text, &kb)
		if err != nil || public {
			if public && id != 0 {
				h.Screen.Delete(ctx, chatID, id)
			}
			if err == nil {
				err = fmt.Errorf("hidden error prompt was public")
			}
			return fmt.Errorf("error screen: %w", err)
		}
		if err := h.Sessions.SetWaiting(ctx, chatID, domain.WaitError, 0, talk.Actor, talk.Name); err != nil {
			return fmt.Errorf("error wait: %w", err)
		}
		if err := h.Sessions.SetDraft(ctx, chatID, "", screen.PromptMark(id, public)); err != nil {
			return fmt.Errorf("error prompt: %w", err)
		}
		h.Screen.Delete(ctx, chatID, userMsgID)
		return nil
	}
	id, err := h.Screen.SendPlain(ctx, chatID, text, &kb)
	if err != nil {
		return fmt.Errorf("error screen: %w", err)
	}
	if err := h.Sessions.SetWaiting(ctx, chatID, domain.WaitError, 0, 0, ""); err != nil {
		h.Screen.Delete(ctx, chatID, id)
		return fmt.Errorf("error wait: %w", err)
	}
	if err := h.Sessions.SetDraft(ctx, chatID, "", id); err != nil {
		h.Screen.Delete(ctx, chatID, id)
		return fmt.Errorf("error prompt: %w", err)
	}
	h.Screen.Delete(ctx, chatID, userMsgID)
	return nil
}

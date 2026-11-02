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
		if _, _, err := h.Screen.Whisper(ctx, chatID, talk, text, &kb); err != nil {
			return fmt.Errorf("error screen: %w", err)
		}
		if err := h.Sessions.SetWaiting(ctx, chatID, domain.WaitError, 0, talk.Actor, talk.Name); err != nil {
			return fmt.Errorf("error wait: %w", err)
		}
		h.Screen.Delete(ctx, chatID, userMsgID)
		return nil
	}
	if err := h.Screen.SendCommand(ctx, chatID, text, kb); err != nil {
		return fmt.Errorf("error screen: %w", err)
	}
	sess, err := h.Sessions.Load(ctx, chatID)
	if err != nil {
		return fmt.Errorf("error load session: %w", err)
	}
	if err := h.Sessions.SetWaiting(ctx, chatID, domain.WaitError, sess.MsgID, 0, ""); err != nil {
		h.Screen.Delete(ctx, chatID, sess.MsgID)
		_ = h.Sessions.SetFunctional(ctx, chatID, 0)
		return fmt.Errorf("error wait: %w", err)
	}
	h.Screen.Delete(ctx, chatID, userMsgID)
	return nil
}

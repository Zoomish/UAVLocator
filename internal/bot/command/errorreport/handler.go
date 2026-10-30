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

func (h *Handler) Command(ctx context.Context, chatID int64, userMsgID int) error {
	if err := h.Sessions.Ensure(ctx, chatID); err != nil {
		return fmt.Errorf("error session: %w", err)
	}
	text, kb := view.ErrorPrompt()
	if err := h.Screen.SendCommand(ctx, chatID, text, kb); err != nil {
		return fmt.Errorf("error screen: %w", err)
	}
	sess, err := h.Sessions.Load(ctx, chatID)
	if err != nil {
		return fmt.Errorf("error load session: %w", err)
	}
	if err := h.Sessions.SetWaiting(ctx, chatID, domain.WaitError, sess.MsgID); err != nil {
		h.Screen.Delete(chatID, sess.MsgID)
		_ = h.Sessions.SetFunctional(ctx, chatID, 0)
		return fmt.Errorf("error wait: %w", err)
	}
	h.Screen.Delete(chatID, userMsgID)
	return nil
}

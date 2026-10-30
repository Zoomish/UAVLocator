package input

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"locatorbpla/internal/bot/command/settings"
	"locatorbpla/internal/bot/screen"
	"locatorbpla/internal/bot/session"
	"locatorbpla/internal/bot/view"
	"locatorbpla/internal/domain"
	"locatorbpla/internal/repository"
)

type UserError struct {
	Text string
}

func (e *UserError) Error() string { return e.Text }

type Router struct {
	Users    *repository.Users
	Sessions *session.Service
	Settings *settings.Handler
	Screen   *screen.Screen
	AdminID  int64
	Bot      *tgbotapi.BotAPI
}

func (r *Router) Handle(ctx context.Context, chatID int64, userMsgID int, text, username string, waiting string) error {
	switch waiting {
	case domain.WaitLocations:
		return r.locations(ctx, chatID, userMsgID, text)
	case domain.WaitError:
		return r.report(ctx, chatID, userMsgID, text, username)
	default:
		return fmt.Errorf("unknown waiting %q", waiting)
	}
}

func (r *Router) locations(ctx context.Context, chatID int64, userMsgID int, text string) error {
	locs := domain.NormalizeLocations(text)
	if len(locs) == 0 {
		r.Screen.Delete(chatID, userMsgID)
		return &UserError{Text: "Введите хотя бы одну локацию через запятую. Инструкция выше остаётся."}
	}
	if err := r.Users.SetLocations(ctx, chatID, locs); err != nil {
		return fmt.Errorf("save locations: %w", err)
	}
	sess, err := r.Sessions.Load(ctx, chatID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}
	if err := r.Settings.Edit(ctx, chatID, sess.MsgID); err != nil {
		return fmt.Errorf("settings after locations: %w", err)
	}
	if err := r.Screen.ClearNoise(ctx, chatID, userMsgID); err != nil {
		return err
	}
	if err := r.Sessions.ClearWaiting(ctx, chatID); err != nil {
		return fmt.Errorf("clear locations wait: %w", err)
	}
	return nil
}

func (r *Router) report(ctx context.Context, chatID int64, userMsgID int, text, username string) error {
	if strings.TrimSpace(text) == "" {
		r.Screen.Delete(chatID, userMsgID)
		return &UserError{Text: "Опишите ошибку текстом. Инструкция выше остаётся."}
	}
	if err := r.notifyAdmin(chatID, username, text); err != nil {
		r.Screen.Delete(chatID, userMsgID)
		return &UserError{Text: "Не удалось отправить сообщение. Инструкция выше остаётся, попробуйте ещё раз."}
	}
	sess, err := r.Sessions.Load(ctx, chatID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}
	if err := r.Screen.ClearNoise(ctx, chatID, userMsgID); err != nil {
		return err
	}
	if err := r.Sessions.ClearWaiting(ctx, chatID); err != nil {
		return fmt.Errorf("clear error wait: %w", err)
	}
	thanks, kb := view.ErrorThanks()
	if err := r.Screen.EditFunctional(ctx, chatID, sess.MsgID, thanks, kb); err != nil {
		return fmt.Errorf("thanks screen: %w", err)
	}
	return nil
}

func (r *Router) notifyAdmin(chatID int64, username, text string) error {
	if r.AdminID == 0 || r.Bot == nil {
		return fmt.Errorf("admin is not configured")
	}
	who := username
	if who == "" {
		who = "без username"
	} else {
		who = "@" + who
	}
	body := fmt.Sprintf("⚠️ Ошибка!\n%s (%d):\n%s", who, chatID, text)
	body = trimRunes(body, 4000)
	msg := tgbotapi.NewMessage(r.AdminID, body)
	if _, err := r.Bot.Send(msg); err != nil {
		return fmt.Errorf("notify admin: %w", err)
	}
	return nil
}

func trimRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	runes := []rune(s)
	return string(runes[:n])
}

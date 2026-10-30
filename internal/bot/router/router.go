package router

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"locatorbpla/internal/bot/callback"
	"locatorbpla/internal/bot/cb"
	"locatorbpla/internal/bot/command/errorreport"
	"locatorbpla/internal/bot/command/settings"
	"locatorbpla/internal/bot/command/start"
	"locatorbpla/internal/bot/input"
	"locatorbpla/internal/bot/screen"
	"locatorbpla/internal/bot/session"
	"locatorbpla/internal/domain"
)

type Alerter interface {
	Infra(ctx context.Context, err error)
}

type Router struct {
	Log      *slog.Logger
	Bot      *tgbotapi.BotAPI
	Screen   *screen.Screen
	Sessions *session.Service
	Start    *start.Handler
	Settings *settings.Handler
	Report   *errorreport.Handler
	Clicks   *callback.Router
	Input    *input.Router
	Alert    Alerter
}

func (r *Router) Run(ctx context.Context) error {
	cmds := tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "start", Description: "Начальное меню"},
		tgbotapi.BotCommand{Command: "settings", Description: "⚙️ Настройки"},
		tgbotapi.BotCommand{Command: "error", Description: "⚠️ Сообщить об ошибке"},
	)
	if _, err := r.Bot.Request(cmds); err != nil {
		return fmt.Errorf("set commands: %w", err)
	}
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := r.Bot.GetUpdatesChan(u)
	defer r.Bot.StopReceivingUpdates()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case upd, ok := <-updates:
			if !ok {
				return fmt.Errorf("telegram updates closed")
			}
			r.safe(ctx, upd)
		}
	}
}

func (r *Router) safe(ctx context.Context, upd tgbotapi.Update) {
	defer func() {
		if rec := recover(); rec != nil {
			err := fmt.Errorf("panic: %v", rec)
			r.Log.Error("update panic", "err", err)
			if r.Alert != nil {
				r.Alert.Infra(ctx, err)
			}
		}
	}()
	if upd.CallbackQuery != nil {
		r.onCallback(ctx, upd)
		return
	}
	if upd.Message != nil {
		r.onMessage(ctx, upd)
	}
}

func (r *Router) onCallback(ctx context.Context, upd tgbotapi.Update) {
	q := upd.CallbackQuery
	if q.Message == nil || q.Message.Chat == nil || q.Message.Chat.Type != "private" {
		r.Screen.Toast(q.ID, "")
		return
	}
	chatID := q.Message.Chat.ID
	sess, err := r.Sessions.Load(ctx, chatID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		r.fail(ctx, chatID, q.ID, err)
		return
	}
	head, _ := cb.Split(q.Data)
	r.Log.Info("callback",
		"telegram_id", chatID,
		"update_id", upd.UpdateID,
		"waiting_for", sess.WaitingFor,
		"callback", head,
	)
	if sess.WaitingFor != "" && head != "cancel" {
		r.Screen.Toast(q.ID, "Сначала отправьте текст или нажмите Отмена")
		return
	}
	err = r.Clicks.Handle(ctx, cb.Query{
		ID:        q.ID,
		Data:      q.Data,
		ChatID:    chatID,
		MessageID: q.Message.MessageID,
	})
	if err = callback.Wrap(err); err != nil {
		r.fail(ctx, chatID, q.ID, err)
	}
}

func (r *Router) onMessage(ctx context.Context, upd tgbotapi.Update) {
	msg := upd.Message
	if msg.Chat == nil || msg.Chat.Type != "private" {
		return
	}
	chatID := msg.Chat.ID
	sess, err := r.Sessions.Load(ctx, chatID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		r.fail(ctx, chatID, "", err)
		return
	}
	r.Log.Info("message",
		"telegram_id", chatID,
		"update_id", upd.UpdateID,
		"waiting_for", sess.WaitingFor,
	)
	if msg.IsCommand() {
		r.onCommand(ctx, msg, sess)
		return
	}
	if sess.WaitingFor == "" {
		r.Screen.Delete(chatID, msg.MessageID)
		return
	}
	username := ""
	if msg.From != nil {
		username = msg.From.UserName
	}
	err = r.Input.Handle(ctx, chatID, msg.MessageID, msg.Text, username, sess.WaitingFor)
	var ue *input.UserError
	if errors.As(err, &ue) {
		if sendErr := r.Screen.SendUserError(ctx, chatID, ue.Text); sendErr != nil {
			r.fail(ctx, chatID, "", sendErr)
		}
		return
	}
	if err != nil {
		r.fail(ctx, chatID, "", err)
	}
}

func (r *Router) onCommand(ctx context.Context, msg *tgbotapi.Message, sess domain.Session) {
	chatID := msg.Chat.ID
	if sess.WaitingFor != "" {
		text := "Бот ждёт " + domain.WaitingLabel(sess.WaitingFor) + ". Команда это не отменяет."
		if err := r.Screen.SendUserError(ctx, chatID, text); err != nil {
			r.fail(ctx, chatID, "", err)
		}
		r.Screen.Delete(chatID, msg.MessageID)
		return
	}
	name, username := "", ""
	if msg.From != nil {
		name = msg.From.FirstName
		username = msg.From.UserName
	}
	var err error
	switch strings.ToLower(msg.Command()) {
	case "start":
		err = r.Start.Command(ctx, chatID, name, username, msg.MessageID)
	case "settings":
		err = r.Settings.Command(ctx, chatID, msg.MessageID)
	case "error":
		err = r.Report.Command(ctx, chatID, msg.MessageID)
	default:
		r.Screen.Delete(chatID, msg.MessageID)
		return
	}
	if err != nil {
		r.fail(ctx, chatID, "", err)
	}
}

func (r *Router) fail(ctx context.Context, chatID int64, callbackID string, err error) {
	r.Log.Error("handler", "err", err, "telegram_id", chatID)
	if r.Alert != nil {
		r.Alert.Infra(ctx, err)
	}
	if callbackID != "" {
		r.Screen.Toast(callbackID, "Что-то пошло не так")
		return
	}
	_ = r.Screen.SendUserError(ctx, chatID, screen.GenericError)
}

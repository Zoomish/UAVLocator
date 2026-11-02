package router

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

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

const shardCount = 8

type Alerter interface {
	Infra(ctx context.Context, err error)
}

type Router struct {
	Log      *slog.Logger
	Screen   *screen.Screen
	Sessions *session.Service
	Start    *start.Handler
	Settings *settings.Handler
	Report   *errorreport.Handler
	Clicks   *callback.Router
	Input    *input.Router
	Alert    Alerter
	shards   [shardCount]chan *models.Update
	once     sync.Once
}

func (r *Router) Handle(ctx context.Context, _ *bot.Bot, upd *models.Update) {
	if upd == nil {
		return
	}
	r.ensure(ctx)
	chatID, ok := openChat(upd)
	if !ok {
		if upd.CallbackQuery != nil {
			r.Screen.Toast(ctx, upd.CallbackQuery.ID, "")
		}
		return
	}
	select {
	case <-ctx.Done():
	case r.shards[int(uint64(chatID)%shardCount)] <- upd:
	}
}

func (r *Router) ensure(ctx context.Context) {
	r.once.Do(func() {
		for i := range r.shards {
			r.shards[i] = make(chan *models.Update, 64)
			go r.loop(ctx, r.shards[i])
		}
	})
}

func (r *Router) loop(ctx context.Context, ch <-chan *models.Update) {
	for {
		select {
		case <-ctx.Done():
			return
		case upd := <-ch:
			r.safe(ctx, upd)
		}
	}
}

func (r *Router) safe(ctx context.Context, upd *models.Update) {
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

func (r *Router) onCallback(ctx context.Context, upd *models.Update) {
	q := upd.CallbackQuery
	chatID, messageID, ephemeralID, ok := callbackTarget(q)
	if !ok {
		r.Screen.Toast(ctx, q.ID, "")
		return
	}
	sess, err := r.Sessions.Load(ctx, chatID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		r.fail(ctx, chatID, q.ID, err)
		return
	}
	head, _ := cb.Split(q.Data)
	group := chatID < 0
	r.Log.Info("callback",
		"telegram_id", chatID,
		"update_id", upd.ID,
		"waiting_for", sess.WaitingFor,
		"callback", head,
	)
	actor, name := person(&q.From)
	if group && sess.WaitingUser != 0 && actor != sess.WaitingUser && blocksWait(q.Data) {
		r.Screen.Toast(ctx, q.ID, screen.Busy(sess.WaitingName, sess.WaitingUser))
		return
	}
	if !group && sess.MsgID != 0 && messageID != 0 && messageID != sess.MsgID {
		r.Screen.Toast(ctx, q.ID, "Меню устарело")
		return
	}
	if !group && sess.WaitingFor != "" && head != "cancel" && head != "confirm" {
		r.Screen.Toast(ctx, q.ID, "Сначала отправьте текст или нажмите Отмена")
		return
	}
	err = r.Clicks.Handle(ctx, cb.Query{
		ID:          q.ID,
		Data:        q.Data,
		ChatID:      chatID,
		MessageID:   messageID,
		EphemeralID: ephemeralID,
		Actor:       actor,
		Name:        name,
		Group:       group,
	})
	if err = callback.Wrap(err); err != nil {
		r.fail(ctx, chatID, q.ID, err)
	}
}

func (r *Router) onMessage(ctx context.Context, upd *models.Update) {
	msg := upd.Message
	chatID := msg.Chat.ID
	sess, err := r.Sessions.Load(ctx, chatID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		r.fail(ctx, chatID, "", err)
		return
	}
	group := msg.Chat.Type == models.ChatTypeGroup || msg.Chat.Type == models.ChatTypeSupergroup
	actor, editor := person(msg.From)
	r.Log.Info("message",
		"telegram_id", chatID,
		"update_id", upd.ID,
		"waiting_for", sess.WaitingFor,
	)
	if group && sess.WaitingFor != "" && sess.WaitingUser != 0 && actor != sess.WaitingUser {
		if _, ok := commandName(msg); ok {
			text := screen.Busy(sess.WaitingName, sess.WaitingUser)
			_, _, err := r.Screen.Whisper(ctx, chatID, screen.Talk{Actor: actor, ReplyEphemeral: msg.EphemeralMessageID}, text, nil)
			if err != nil {
				r.Screen.SendNote(ctx, chatID, text)
			}
		}
		return
	}
	if name, ok := commandName(msg); ok {
		r.onCommand(ctx, msg, name, sess, screen.Talk{Actor: actor, Name: editor, ReplyEphemeral: msg.EphemeralMessageID})
		return
	}
	if sess.WaitingFor == "" {
		if !group {
			r.Screen.Delete(ctx, chatID, msg.ID)
		}
		return
	}
	username := ""
	if msg.From != nil {
		username = msg.From.Username
	}
	err = r.Input.Handle(ctx, input.Incoming{
		ChatID:    chatID,
		UserMsgID: msg.ID,
		Text:      msg.Text,
		Username:  username,
		Editor:    editor,
		Waiting:   sess.WaitingFor,
		Actor:     actor,
		Ephemeral: msg.EphemeralMessageID,
	})
	var ue *input.UserError
	if errors.As(err, &ue) {
		if group && actor != 0 {
			if _, _, sendErr := r.Screen.Whisper(ctx, chatID, screen.Talk{Actor: actor, ReplyEphemeral: msg.EphemeralMessageID, AllowPublic: true}, ue.Text, nil); sendErr != nil {
				r.fail(ctx, chatID, "", sendErr)
			}
			return
		}
		if sendErr := r.Screen.SendUserError(ctx, chatID, ue.Text); sendErr != nil {
			r.fail(ctx, chatID, "", sendErr)
		}
		return
	}
	if err != nil {
		r.fail(ctx, chatID, "", err)
	}
}

func (r *Router) onCommand(ctx context.Context, msg *models.Message, name string, sess domain.Session, talk screen.Talk) {
	chatID := msg.Chat.ID
	group := chatID < 0
	if name == "start" {
		released, relErr := r.Start.Users.ReleaseBlock(ctx, chatID)
		if relErr != nil {
			r.fail(ctx, chatID, "", relErr)
			return
		}
		if released && sess.WaitingFor != "" {
			if err := r.Input.Rollback(ctx, chatID); err != nil {
				r.fail(ctx, chatID, "", err)
			}
			r.Screen.Delete(ctx, chatID, msg.ID)
			return
		}
	}
	if sess.WaitingFor != "" {
		text := "Бот ждёт " + domain.WaitingLabel(sess.WaitingFor) + ". Команда это не отменяет."
		if group && talk.Actor != 0 {
			talk.AllowPublic = true
			if _, _, err := r.Screen.Whisper(ctx, chatID, talk, text, nil); err != nil {
				r.fail(ctx, chatID, "", err)
			}
		} else if err := r.Screen.SendUserError(ctx, chatID, text); err != nil {
			r.fail(ctx, chatID, "", err)
		}
		r.Screen.Delete(ctx, chatID, msg.ID)
		return
	}
	first, username := "", ""
	if msg.From != nil {
		first = msg.From.FirstName
		username = msg.From.Username
	}
	if group {
		first = msg.Chat.Title
		username = ""
	}
	var err error
	switch name {
	case "start":
		err = r.Start.Command(ctx, chatID, first, username, msg.ID, group)
	case "settings":
		err = r.Settings.Command(ctx, chatID, msg.ID, talk)
	case "error":
		err = r.Report.Command(ctx, chatID, msg.ID, talk)
	default:
		r.Screen.Delete(ctx, chatID, msg.ID)
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
		r.Screen.Toast(ctx, callbackID, "Что-то пошло не так")
		return
	}
	_ = r.Screen.SendUserError(ctx, chatID, screen.GenericError)
}

func openChat(upd *models.Update) (int64, bool) {
	if upd.Message != nil && chatAllowed(upd.Message.Chat.Type) {
		return upd.Message.Chat.ID, true
	}
	if upd.CallbackQuery == nil {
		return 0, false
	}
	chatID, _, _, ok := callbackTarget(upd.CallbackQuery)
	return chatID, ok
}

func chatAllowed(t models.ChatType) bool {
	switch t {
	case models.ChatTypePrivate, models.ChatTypeGroup, models.ChatTypeSupergroup:
		return true
	default:
		return false
	}
}

func callbackTarget(q *models.CallbackQuery) (int64, int, int, bool) {
	if q == nil {
		return 0, 0, 0, false
	}
	switch q.Message.Type {
	case models.MaybeInaccessibleMessageTypeMessage:
		if q.Message.Message == nil || !chatAllowed(q.Message.Message.Chat.Type) {
			return 0, 0, 0, false
		}
		msg := q.Message.Message
		return msg.Chat.ID, msg.ID, msg.EphemeralMessageID, true
	case models.MaybeInaccessibleMessageTypeInaccessibleMessage:
		if q.Message.InaccessibleMessage == nil || !chatAllowed(q.Message.InaccessibleMessage.Chat.Type) {
			return 0, 0, 0, false
		}
		msg := q.Message.InaccessibleMessage
		return msg.Chat.ID, msg.MessageID, 0, true
	default:
		return 0, 0, 0, false
	}
}

func blocksWait(data string) bool {
	switch data {
	case "set-locations", "get-api", "set-api", "set-api-off", "cancel-locations", "cancel-api", "cancel-webhook", "cancel-error", "confirm-save", "confirm-again":
		return true
	default:
		return false
	}
}

func person(from *models.User) (int64, string) {
	if from == nil {
		return 0, ""
	}
	if from.Username != "" {
		return from.ID, "@" + from.Username
	}
	if from.FirstName != "" {
		return from.ID, from.FirstName
	}
	return from.ID, strconv.FormatInt(from.ID, 10)
}

func commandName(msg *models.Message) (string, bool) {
	if msg == nil || msg.Text == "" {
		return "", false
	}
	for _, entity := range msg.Entities {
		if entity.Type != models.MessageEntityTypeBotCommand || entity.Offset != 0 {
			continue
		}
		raw := entityText(msg.Text, entity.Offset, entity.Length)
		raw = strings.TrimPrefix(raw, "/")
		if i := strings.IndexByte(raw, '@'); i >= 0 {
			raw = raw[:i]
		}
		raw = strings.ToLower(raw)
		return raw, raw != ""
	}
	return "", false
}

func entityText(text string, offset, length int) string {
	units := utf16.Encode([]rune(text))
	if offset < 0 || length < 0 || offset+length > len(units) {
		return ""
	}
	return string(utf16.Decode(units[offset : offset+length]))
}

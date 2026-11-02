package input

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"locatorbpla/internal/bot/cb"
	"locatorbpla/internal/bot/command/settings"
	"locatorbpla/internal/bot/screen"
	"locatorbpla/internal/bot/session"
	"locatorbpla/internal/bot/view"
	"locatorbpla/internal/domain"
	"locatorbpla/internal/notify"
	"locatorbpla/internal/repository"
)

type Incoming struct {
	ChatID    int64
	UserMsgID int
	Text      string
	Username  string
	Editor    string
	Waiting   string
	Actor     int64
	Ephemeral int
}

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
	Bot      *bot.Bot
}

func (r *Router) Handle(ctx context.Context, in Incoming) error {
	switch in.Waiting {
	case domain.WaitLocations:
		return r.locations(ctx, in)
	case domain.WaitError:
		return r.report(ctx, in)
	case domain.WaitWebhook:
		return r.webhook(ctx, in)
	default:
		return fmt.Errorf("unknown waiting %q", in.Waiting)
	}
}

func (r *Router) locations(ctx context.Context, in Incoming) error {
	locs := domain.NormalizeLocations(in.Text)
	if len(locs) == 0 {
		r.Screen.Delete(ctx, in.ChatID, in.UserMsgID)
		return &UserError{Text: "Введите хотя бы одну локацию через запятую. Инструкция выше остаётся."}
	}
	return r.ask(ctx, in, strings.Join(locs, ", "))
}

func (r *Router) commitLocations(ctx context.Context, in Incoming, phrase string) error {
	locs := domain.NormalizeLocations(phrase)
	if len(locs) == 0 {
		return &UserError{Text: "Введите хотя бы одну локацию через запятую."}
	}
	editorID := int64(0)
	editor := ""
	if in.ChatID < 0 {
		editorID = in.Actor
		editor = in.Editor
	}
	if err := r.Users.SetLocations(ctx, in.ChatID, locs, editorID, editor); err != nil {
		return fmt.Errorf("save locations: %w", err)
	}
	sess, err := r.Sessions.Load(ctx, in.ChatID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}
	if err := r.Sessions.ClearWaiting(ctx, in.ChatID); err != nil {
		return fmt.Errorf("clear locations wait: %w", err)
	}
	if in.ChatID < 0 {
		user, err := r.Users.Find(ctx, in.ChatID)
		if err != nil {
			return fmt.Errorf("group after locations: %w", err)
		}
		text, kb := view.GroupBoard(user)
		if err := r.showBoard(ctx, in.ChatID, sess.MsgID, text, kb); err != nil {
			return err
		}
		r.Screen.SendNote(ctx, in.ChatID, "Локации изменил "+who(in.Editor, in.Actor))
		return nil
	}
	if err := r.Settings.Edit(ctx, in.ChatID, sess.MsgID); err != nil {
		if !screen.Gone(err) {
			return fmt.Errorf("settings after locations: %w", err)
		}
		user, ferr := r.Users.Find(ctx, in.ChatID)
		if ferr != nil {
			return fmt.Errorf("settings after locations: %w", ferr)
		}
		text, kb := view.Settings(user, true)
		if err := r.Screen.SendCommand(ctx, in.ChatID, text, kb); err != nil {
			return fmt.Errorf("settings after locations: %w", err)
		}
	}
	return r.Screen.ClearNoise(ctx, in.ChatID, in.UserMsgID)
}

func (r *Router) webhook(ctx context.Context, in Incoming) error {
	if in.ChatID < 0 {
		_ = r.Sessions.ClearWaiting(ctx, in.ChatID)
		if in.Ephemeral != 0 {
			r.Screen.DeleteEphemeral(ctx, in.ChatID, in.Actor, in.Ephemeral)
		} else {
			r.Screen.Delete(ctx, in.ChatID, in.UserMsgID)
		}
		return nil
	}
	raw := strings.TrimSpace(in.Text)
	if raw != "-" {
		if err := notify.PublicHTTPS(ctx, raw); err != nil {
			if in.Ephemeral == 0 {
				r.Screen.Delete(ctx, in.ChatID, in.UserMsgID)
			}
			return &UserError{Text: "Нужен публичный адрес https. «-» стирает адрес. Инструкция выше остаётся."}
		}
	} else {
		raw = ""
	}
	if err := r.Users.SetWebhook(ctx, in.ChatID, raw); err != nil {
		return fmt.Errorf("save webhook: %w", err)
	}
	sess, err := r.Sessions.Load(ctx, in.ChatID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}
	if err := r.Sessions.ClearWaiting(ctx, in.ChatID); err != nil {
		return fmt.Errorf("clear webhook wait: %w", err)
	}
	user, err := r.Users.Find(ctx, in.ChatID)
	if err != nil {
		return fmt.Errorf("api user: %w", err)
	}
	text, kb := view.APIMenu(user)
	err = r.Screen.EditFunctional(ctx, in.ChatID, sess.MsgID, text, kb)
	if err != nil && !screen.Gone(err) {
		return fmt.Errorf("api menu: %w", err)
	}
	if screen.Gone(err) {
		if err = r.Screen.SendCommand(ctx, in.ChatID, text, kb); err != nil {
			return fmt.Errorf("api menu: %w", err)
		}
	}
	if err := r.Screen.ClearNoise(ctx, in.ChatID, in.UserMsgID); err != nil {
		return err
	}
	return nil
}

func (r *Router) showBoard(ctx context.Context, chatID int64, msgID int, text string, kb models.InlineKeyboardMarkup) error {
	if msgID == 0 {
		if err := r.Screen.SendCommand(ctx, chatID, text, kb); err != nil {
			return fmt.Errorf("group board: %w", err)
		}
		return nil
	}
	err := r.Screen.EditFunctional(ctx, chatID, msgID, text, kb)
	if err == nil {
		return nil
	}
	if !screen.Gone(err) {
		return fmt.Errorf("group board: %w", err)
	}
	if err := r.Screen.SendCommand(ctx, chatID, text, kb); err != nil {
		return fmt.Errorf("group board: %w", err)
	}
	return nil
}

func (r *Router) preview(ctx context.Context, in Incoming, phrase, text string, kb models.InlineKeyboardMarkup) error {
	sess, err := r.Sessions.Load(ctx, in.ChatID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}
	if in.ChatID < 0 && in.Ephemeral != 0 {
		body := view.HiddenIntro(in.Editor, false) + text
		id, _, err := r.Screen.Whisper(ctx, in.ChatID, screen.Talk{Actor: in.Actor, ReplyEphemeral: in.Ephemeral}, body, &kb)
		if err != nil {
			return fmt.Errorf("confirm screen: %w", err)
		}
		if err := r.Sessions.SetDraft(ctx, in.ChatID, phrase, id); err != nil {
			return err
		}
		r.Screen.DeleteEphemeral(ctx, in.ChatID, in.Actor, in.Ephemeral)
		return nil
	}
	if in.ChatID < 0 {
		msgID := sess.DraftMsg
		if msgID != 0 {
			if err := r.Screen.EditPlain(ctx, in.ChatID, msgID, text, kb); err == nil {
				if err := r.Sessions.SetDraft(ctx, in.ChatID, phrase, 0); err != nil {
					return err
				}
				r.Screen.Delete(ctx, in.ChatID, in.UserMsgID)
				return nil
			}
		}
		id, err := r.Screen.SendPlain(ctx, in.ChatID, text, &kb)
		if err != nil {
			return fmt.Errorf("confirm screen: %w", err)
		}
		if err := r.Sessions.SetDraft(ctx, in.ChatID, phrase, id); err != nil {
			return err
		}
		r.Screen.Delete(ctx, in.ChatID, in.UserMsgID)
		return nil
	}
	if err := r.Screen.EditFunctional(ctx, in.ChatID, sess.MsgID, text, kb); err != nil {
		if !screen.Gone(err) {
			return fmt.Errorf("confirm screen: %w", err)
		}
		if err := r.Screen.SendCommand(ctx, in.ChatID, text, kb); err != nil {
			return fmt.Errorf("confirm screen: %w", err)
		}
	}
	if err := r.Sessions.SetDraft(ctx, in.ChatID, phrase, 0); err != nil {
		return err
	}
	return r.Screen.ClearNoise(ctx, in.ChatID, in.UserMsgID)
}

func (r *Router) Confirm(ctx context.Context, q cb.Query, action string) error {
	sess, err := r.Sessions.Load(ctx, q.ChatID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}
	if q.Group && sess.WaitingUser != 0 && q.Actor != sess.WaitingUser {
		r.Screen.Toast(ctx, q.ID, screen.Busy(sess.WaitingName, sess.WaitingUser))
		return nil
	}
	if action == "again" {
		if err := r.Sessions.SetDraft(ctx, q.ChatID, "", 0); err != nil {
			return err
		}
		return r.repaint(ctx, q, sess.WaitingFor, true)
	}
	if sess.Draft == "" {
		r.Screen.Toast(ctx, q.ID, "Нечего сохранять")
		return nil
	}
	in := Incoming{
		ChatID: q.ChatID, Text: sess.Draft, Username: strings.TrimPrefix(q.Name, "@"),
		Editor: q.Name, Actor: q.Actor, Ephemeral: q.EphemeralID,
	}
	switch sess.WaitingFor {
	case domain.WaitLocations:
		if q.Group {
			ok, err := r.Screen.IsChatAdmin(ctx, q.ChatID, q.Actor)
			if err != nil || !ok {
				r.Screen.Toast(ctx, q.ID, "Недостаточно прав")
				return nil
			}
		}
		if err := r.commitLocations(ctx, in, sess.Draft); err != nil {
			return err
		}
	case domain.WaitError:
		if err := r.commitReport(ctx, in, sess.Draft); err != nil {
			return err
		}
	default:
		r.Screen.Toast(ctx, q.ID, "Эта кнопка уже не действует")
		return nil
	}
	if q.Group && q.EphemeralID != 0 {
		r.Screen.DeleteEphemeral(ctx, q.ChatID, q.Actor, q.EphemeralID)
	} else if q.Group && q.MessageID != 0 && q.MessageID != sess.MsgID {
		r.Screen.Delete(ctx, q.ChatID, q.MessageID)
	}
	r.Screen.Toast(ctx, q.ID, "")
	return nil
}

func (r *Router) repaint(ctx context.Context, q cb.Query, waiting string, ask bool) error {
	var text string
	var kb models.InlineKeyboardMarkup
	switch waiting {
	case domain.WaitLocations:
		text, kb = view.LocationsPrompt()
	case domain.WaitError:
		text, kb = view.ErrorPrompt()
	default:
		r.Screen.Toast(ctx, q.ID, "Эта кнопка уже не действует")
		return nil
	}
	if q.Group && q.EphemeralID != 0 {
		body := view.HiddenIntro(q.Name, ask) + text
		_, _, err := r.Screen.Whisper(ctx, q.ChatID, screen.Talk{
			Actor: q.Actor, CallbackID: q.ID, EphemeralID: q.EphemeralID,
		}, body, &kb)
		if err != nil {
			h := text
			if _, plainErr := r.Screen.SendPlain(ctx, q.ChatID, h, &kb); plainErr != nil {
				return fmt.Errorf("prompt: %w", err)
			}
		}
		r.Screen.Toast(ctx, q.ID, "")
		return nil
	}
	if q.Group {
		if err := r.Screen.EditPlain(ctx, q.ChatID, q.MessageID, text, kb); err != nil {
			if _, plainErr := r.Screen.SendPlain(ctx, q.ChatID, text, &kb); plainErr != nil {
				return fmt.Errorf("prompt: %w", err)
			}
		}
		r.Screen.Toast(ctx, q.ID, "")
		return nil
	}
	if err := r.Screen.EditFunctional(ctx, q.ChatID, q.MessageID, text, kb); err != nil {
		return fmt.Errorf("prompt: %w", err)
	}
	r.Screen.Toast(ctx, q.ID, "")
	return nil
}

func (r *Router) Rollback(ctx context.Context, chatID int64) error {
	sess, err := r.Sessions.Load(ctx, chatID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("rollback session: %w", err)
	}
	waiting := sess.WaitingFor
	msgID := sess.MsgID
	if err := r.Sessions.ClearWaiting(ctx, chatID); err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	if waiting == "" {
		return nil
	}
	if chatID < 0 {
		user, err := r.Users.Find(ctx, chatID)
		if err != nil {
			return fmt.Errorf("rollback group: %w", err)
		}
		text, kb := view.GroupBoard(user)
		return r.showBoard(ctx, chatID, msgID, text, kb)
	}
	user, err := r.Users.Find(ctx, chatID)
	if err != nil {
		return fmt.Errorf("rollback user: %w", err)
	}
	var text string
	var kb models.InlineKeyboardMarkup
	if waiting == domain.WaitWebhook {
		text, kb = view.APIMenu(user)
	} else {
		text, kb = view.Settings(user, true)
	}
	if err := r.Screen.EditFunctional(ctx, chatID, msgID, text, kb); err != nil {
		if !screen.Gone(err) {
			return err
		}
		return r.Screen.SendCommand(ctx, chatID, text, kb)
	}
	return nil
}

func who(editor string, id int64) string {
	if editor != "" {
		return html.EscapeString(editor)
	}
	if id == 0 {
		return "кто-то"
	}
	return strconv.FormatInt(id, 10)
}

func (r *Router) report(ctx context.Context, in Incoming) error {
	phrase := strings.TrimSpace(in.Text)
	if phrase == "" {
		r.Screen.Delete(ctx, in.ChatID, in.UserMsgID)
		return &UserError{Text: "Опишите ошибку текстом. Инструкция выше остаётся."}
	}
	return r.ask(ctx, in, phrase)
}

func (r *Router) commitReport(ctx context.Context, in Incoming, phrase string) error {
	if strings.TrimSpace(phrase) == "" {
		return &UserError{Text: "Опишите ошибку текстом."}
	}
	if err := r.notifyAdmin(ctx, in.ChatID, in.Username, phrase); err != nil {
		return &UserError{Text: "Не удалось отправить сообщение. Попробуйте ещё раз."}
	}
	sess, err := r.Sessions.Load(ctx, in.ChatID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}
	if err := r.Sessions.ClearWaiting(ctx, in.ChatID); err != nil {
		return fmt.Errorf("clear error wait: %w", err)
	}
	thanks, kb := view.ErrorThanks()
	if in.ChatID < 0 {
		user, err := r.Users.Find(ctx, in.ChatID)
		if err != nil {
			return fmt.Errorf("group after report: %w", err)
		}
		board, boardKB := view.GroupBoard(user)
		return r.showBoard(ctx, in.ChatID, sess.MsgID, board, boardKB)
	}
	err = r.Screen.EditFunctional(ctx, in.ChatID, sess.MsgID, thanks, kb)
	if err != nil && !screen.Gone(err) {
		return fmt.Errorf("thanks screen: %w", err)
	}
	if screen.Gone(err) {
		if err = r.Screen.SendCommand(ctx, in.ChatID, thanks, kb); err != nil {
			return fmt.Errorf("thanks screen: %w", err)
		}
	}
	return nil
}

func (r *Router) notifyAdmin(ctx context.Context, chatID int64, username, text string) error {
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
	if _, err := r.Bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: r.AdminID,
		Text:   body,
	}); err != nil {
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

package screen

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"locatorbpla/internal/bot/session"
)

const GenericError = "Что-то пошло не так. Попробуйте позже."

type Screen struct {
	Bot      *bot.Bot
	Sessions *session.Service
	Denied   func(context.Context, int64)
}

func (s *Screen) SendCommand(ctx context.Context, chatID int64, text string, kb models.InlineKeyboardMarkup) error {
	prev := 0
	if sess, err := s.Sessions.Load(ctx, chatID); err == nil {
		prev = sess.MsgID
	}
	id, err := s.send(ctx, chatID, text, &kb)
	if err != nil {
		return err
	}
	if err := s.Sessions.SetFunctional(ctx, chatID, id); err != nil {
		return fmt.Errorf("store command message: %w", err)
	}
	if prev != 0 && prev != id {
		s.Delete(ctx, chatID, prev)
	}
	return nil
}

func (s *Screen) SendPlain(ctx context.Context, chatID int64, text string, kb *models.InlineKeyboardMarkup) (int, error) {
	return s.send(ctx, chatID, text, kb)
}

func (s *Screen) EditFunctional(ctx context.Context, chatID int64, messageID int, text string, kb models.InlineKeyboardMarkup) error {
	if messageID == 0 {
		return s.SendCommand(ctx, chatID, text, kb)
	}
	if err := s.edit(ctx, chatID, messageID, text, kb); err != nil {
		return err
	}
	if err := s.Sessions.SetFunctional(ctx, chatID, messageID); err != nil {
		return fmt.Errorf("store edited message: %w", err)
	}
	return nil
}

func (s *Screen) SendUserError(ctx context.Context, chatID int64, text string) error {
	sess, err := s.Sessions.Load(ctx, chatID)
	if err != nil {
		return fmt.Errorf("load session for error: %w", err)
	}
	if sess.ErrorMsgID != 0 {
		s.Delete(ctx, chatID, sess.ErrorMsgID)
	}
	id, err := s.send(ctx, chatID, text, nil)
	if err != nil {
		return err
	}
	if err := s.Sessions.SetError(ctx, chatID, id); err != nil {
		return fmt.Errorf("store error message: %w", err)
	}
	return nil
}

func (s *Screen) ClearNoise(ctx context.Context, chatID int64, userMsgID int) error {
	sess, err := s.Sessions.Load(ctx, chatID)
	if err != nil {
		return fmt.Errorf("load session for cleanup: %w", err)
	}
	if userMsgID != 0 {
		s.Delete(ctx, chatID, userMsgID)
	}
	if sess.ErrorMsgID != 0 {
		s.Delete(ctx, chatID, sess.ErrorMsgID)
	}
	if err := s.Sessions.SetError(ctx, chatID, 0); err != nil {
		return fmt.Errorf("clear error message: %w", err)
	}
	return nil
}

func (s *Screen) Delete(ctx context.Context, chatID int64, messageID int) {
	if messageID == 0 || s.Bot == nil {
		return
	}
	_, _ = s.Bot.DeleteMessage(ctx, &bot.DeleteMessageParams{ChatID: chatID, MessageID: messageID})
}

type Talk struct {
	Actor          int64
	Name           string
	CallbackID     string
	EphemeralID    int
	ReplyEphemeral int
	Replace        bool
	AllowPublic    bool
}

func (s *Screen) Whisper(ctx context.Context, chatID int64, talk Talk, text string, kb *models.InlineKeyboardMarkup) (int, bool, error) {
	if talk.EphemeralID != 0 && talk.ReplyEphemeral == 0 {
		err := s.editEphemeral(ctx, chatID, talk.Actor, talk.EphemeralID, text, kb)
		if err == nil {
			s.Toast(ctx, talk.CallbackID, "")
			return talk.EphemeralID, false, nil
		}
		if !talk.AllowPublic {
			return 0, false, err
		}
		id, ferr := s.send(ctx, chatID, text, kb)
		if ferr != nil {
			return 0, true, err
		}
		return id, true, nil
	}
	params := &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
		EphemeralMessageParameters: &models.EphemeralMessageParameters{
			ReceiverUserID:              talk.Actor,
			CallbackQueryID:             talk.CallbackID,
			ReplaceCallbackQueryMessage: talk.Replace && talk.CallbackID != "" && talk.ReplyEphemeral == 0,
		},
	}
	if talk.ReplyEphemeral != 0 {
		params.ReplyParameters = &models.ReplyParameters{EphemeralMessageID: talk.ReplyEphemeral}
		params.EphemeralMessageParameters.CallbackQueryID = ""
		params.EphemeralMessageParameters.ReplaceCallbackQueryMessage = false
	}
	if kb != nil {
		params.ReplyMarkup = kb
	}
	sent, err := s.Bot.SendMessage(ctx, params)
	if err == nil {
		id := sent.EphemeralMessageID
		if id == 0 {
			id = sent.ID
		}
		return id, false, nil
	}
	s.forbid(ctx, chatID, err)
	if denied(err) || !talk.AllowPublic {
		return 0, false, fmt.Errorf("send ephemeral: %w", err)
	}
	id, ferr := s.send(ctx, chatID, text, kb)
	if ferr != nil {
		return 0, true, fmt.Errorf("send ephemeral: %w", err)
	}
	return id, true, nil
}

func (s *Screen) editEphemeral(ctx context.Context, chatID, actor int64, ephemeralID int, text string, kb *models.InlineKeyboardMarkup) error {
	params := &bot.EditEphemeralMessageTextParams{
		ChatID:             chatID,
		ReceiverUserID:     actor,
		EphemeralMessageID: ephemeralID,
		Text:               text,
		ParseMode:          models.ParseModeHTML,
		ReplyMarkup:        kb,
	}
	_, err := s.Bot.EditEphemeralMessageText(ctx, params)
	if err != nil && errors.Is(err, bot.ErrorBadRequest) && strings.Contains(err.Error(), "message is not modified") {
		return nil
	}
	if err != nil {
		s.forbid(ctx, chatID, err)
		return fmt.Errorf("edit ephemeral: %w", err)
	}
	return nil
}

func (s *Screen) DeleteEphemeral(ctx context.Context, chatID, actor int64, ephemeralID int) {
	if ephemeralID == 0 || actor == 0 || s.Bot == nil {
		return
	}
	_, _ = s.Bot.DeleteEphemeralMessage(ctx, &bot.DeleteEphemeralMessageParams{
		ChatID:             chatID,
		ReceiverUserID:     actor,
		EphemeralMessageID: ephemeralID,
	})
}

func (s *Screen) SendNote(ctx context.Context, chatID int64, text string) {
	if s.Bot == nil || text == "" {
		return
	}
	_, _ = s.send(ctx, chatID, text, nil)
}

func (s *Screen) IsChatAdmin(ctx context.Context, chatID, userID int64) (bool, error) {
	if chatID > 0 {
		return true, nil
	}
	if s.Bot == nil || userID == 0 {
		return false, fmt.Errorf("chat member: missing bot or user")
	}
	member, err := s.Bot.GetChatMember(ctx, &bot.GetChatMemberParams{ChatID: chatID, UserID: userID})
	if err != nil {
		return false, fmt.Errorf("chat member: %w", err)
	}
	switch member.Type {
	case models.ChatMemberTypeOwner, models.ChatMemberTypeAdministrator:
		return true, nil
	default:
		return false, nil
	}
}

func (s *Screen) forbid(ctx context.Context, chatID int64, err error) {
	if s == nil || s.Denied == nil || !denied(err) {
		return
	}
	s.Denied(ctx, chatID)
}

func denied(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, bot.ErrorForbidden) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "bot was blocked") || strings.Contains(msg, "user is deactivated") || strings.Contains(msg, "Forbidden") || strings.Contains(msg, "bot was kicked")
}

func Busy(name string, id int64) string {
	if name == "" {
		name = fmt.Sprintf("%d", id)
	}
	return "Сейчас вводит " + name
}

func Gone(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "message to edit not found") ||
		strings.Contains(msg, "message can't be edited") ||
		strings.Contains(msg, "MESSAGE_ID_INVALID")
}

func (s *Screen) Toast(ctx context.Context, callbackID, text string) {
	if callbackID == "" || s.Bot == nil {
		return
	}
	_, _ = s.Bot.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callbackID,
		Text:            text,
	})
}

func (s *Screen) send(ctx context.Context, chatID int64, text string, kb *models.InlineKeyboardMarkup) (int, error) {
	params := &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	}
	if kb != nil {
		params.ReplyMarkup = kb
	}
	sent, err := s.Bot.SendMessage(ctx, params)
	if err != nil {
		s.forbid(ctx, chatID, err)
		return 0, fmt.Errorf("send message: %w", err)
	}
	return sent.ID, nil
}

func (s *Screen) EditPlain(ctx context.Context, chatID int64, messageID int, text string, kb models.InlineKeyboardMarkup) error {
	return s.edit(ctx, chatID, messageID, text, kb)
}

func (s *Screen) edit(ctx context.Context, chatID int64, messageID int, text string, kb models.InlineKeyboardMarkup) error {
	_, err := s.Bot.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: kb,
	})
	if err != nil && errors.Is(err, bot.ErrorBadRequest) && strings.Contains(err.Error(), "message is not modified") {
		return nil
	}
	if err != nil {
		s.forbid(ctx, chatID, err)
		return fmt.Errorf("edit message: %w", err)
	}
	return nil
}

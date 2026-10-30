package screen

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"locatorbpla/internal/bot/session"
)

const GenericError = "Что-то пошло не так. Попробуйте позже."

type Screen struct {
	Bot      *tgbotapi.BotAPI
	Sessions *session.Service
}

func (s *Screen) SendCommand(ctx context.Context, chatID int64, text string, kb tgbotapi.InlineKeyboardMarkup) error {
	id, err := s.send(chatID, text, &kb)
	if err != nil {
		return err
	}
	if err := s.Sessions.SetFunctional(ctx, chatID, id); err != nil {
		return fmt.Errorf("store command message: %w", err)
	}
	return nil
}

func (s *Screen) EditFunctional(ctx context.Context, chatID int64, messageID int, text string, kb tgbotapi.InlineKeyboardMarkup) error {
	if messageID == 0 {
		return s.SendCommand(ctx, chatID, text, kb)
	}
	if err := s.edit(chatID, messageID, text, kb); err != nil {
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
		s.Delete(chatID, sess.ErrorMsgID)
	}
	id, err := s.send(chatID, text, nil)
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
		s.Delete(chatID, userMsgID)
	}
	if sess.ErrorMsgID != 0 {
		s.Delete(chatID, sess.ErrorMsgID)
	}
	if err := s.Sessions.SetError(ctx, chatID, 0); err != nil {
		return fmt.Errorf("clear error message: %w", err)
	}
	return nil
}

func (s *Screen) Delete(chatID int64, messageID int) {
	if messageID == 0 {
		return
	}
	_, _ = s.Bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
}

func (s *Screen) Toast(callbackID, text string) {
	cb := tgbotapi.NewCallback(callbackID, text)
	_, _ = s.Bot.Request(cb)
}

func (s *Screen) send(chatID int64, text string, kb *tgbotapi.InlineKeyboardMarkup) (int, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"
	if kb != nil {
		msg.ReplyMarkup = kb
	}
	sent, err := s.Bot.Send(msg)
	if err != nil {
		return 0, fmt.Errorf("send message: %w", err)
	}
	return sent.MessageID, nil
}

func (s *Screen) edit(chatID int64, messageID int, text string, kb tgbotapi.InlineKeyboardMarkup) error {
	edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, kb)
	edit.ParseMode = "HTML"
	_, err := s.Bot.Send(edit)
	if err != nil && strings.Contains(err.Error(), "message is not modified") {
		return nil
	}
	if err != nil {
		return fmt.Errorf("edit message: %w", err)
	}
	return nil
}

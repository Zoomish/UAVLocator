package notify

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"locatorbpla/internal/domain"
)

type Disabler interface {
	SetNotifications(ctx context.Context, tgID int64, on bool) error
}

type Sender struct {
	Bot   *tgbotapi.BotAPI
	Users Disabler
	Log   *slog.Logger
	Gap   time.Duration
}

func (s *Sender) Broadcast(ctx context.Context, html string, users []domain.User) error {
	gap := s.Gap
	if gap <= 0 {
		gap = 40 * time.Millisecond
	}
	for i, user := range users {
		if err := s.Send(ctx, user.TgID, html); err != nil {
			return err
		}
		if i == len(users)-1 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(gap):
		}
	}
	return nil
}

func (s *Sender) Send(ctx context.Context, tgID int64, html string) error {
	msg := tgbotapi.NewMessage(tgID, html)
	msg.ParseMode = "HTML"
	_, err := s.Bot.Send(msg)
	if err == nil {
		return nil
	}
	if wait, ok := retryAfter(err); ok {
		return &RetryAfterError{Wait: wait}
	}
	if blocked(err) {
		if offErr := s.Users.SetNotifications(ctx, tgID, false); offErr != nil {
			return fmt.Errorf("disable notifications for %d: %w", tgID, offErr)
		}
		s.Log.Warn("notifications disabled", "telegram_id", tgID, "err", err)
		return nil
	}
	s.Log.Error("send alert", "telegram_id", tgID, "err", err)
	return fmt.Errorf("send alert to %d: %w", tgID, err)
}

func blocked(err error) bool {
	var apiErr *tgbotapi.Error
	if errors.As(err, &apiErr) && apiErr != nil && apiErr.Code == 403 {
		return true
	}
	var apiVal tgbotapi.Error
	if errors.As(err, &apiVal) && apiVal.Code == 403 {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "bot was blocked") || strings.Contains(msg, "user is deactivated") || strings.Contains(msg, "Forbidden")
}

type RetryAfterError struct {
	Wait time.Duration
}

func (e *RetryAfterError) Error() string {
	return "telegram flood wait"
}

func retryAfter(err error) (time.Duration, bool) {
	var apiErr *tgbotapi.Error
	if errors.As(err, &apiErr) && apiErr != nil && apiErr.Code == 429 && apiErr.RetryAfter > 0 {
		return time.Duration(apiErr.RetryAfter) * time.Second, true
	}
	var apiVal tgbotapi.Error
	if errors.As(err, &apiVal) && apiVal.Code == 429 && apiVal.RetryAfter > 0 {
		return time.Duration(apiVal.RetryAfter) * time.Second, true
	}
	return 0, false
}

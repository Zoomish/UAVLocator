package ops

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Alerter struct {
	Bot     *tgbotapi.BotAPI
	AdminID int64
	Log     *slog.Logger
}

func (a *Alerter) Infra(ctx context.Context, err error) {
	if err == nil {
		return
	}
	a.Log.Error("infra", "err", err)
	if a.Bot == nil || a.AdminID == 0 {
		a.Log.Error("admin notify skipped", "err", err)
		return
	}
	text := "Ошибка бота:\n" + err.Error()
	if len(text) > 4000 {
		text = text[:4000]
	}
	msg := tgbotapi.NewMessage(a.AdminID, text)
	if _, sendErr := a.Bot.Send(msg); sendErr != nil {
		a.Log.Error("admin notify failed", "err", sendErr, "cause", err.Error())
	}
	_ = ctx
}

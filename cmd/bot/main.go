package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/sync/errgroup"

	"locatorbpla/internal/bot/callback"
	"locatorbpla/internal/bot/callback/admin"
	"locatorbpla/internal/bot/callback/cancel"
	"locatorbpla/internal/bot/callback/confirm"
	"locatorbpla/internal/bot/callback/get"
	"locatorbpla/internal/bot/callback/set/hook"
	"locatorbpla/internal/bot/callback/set/locations"
	"locatorbpla/internal/bot/callback/set/notifications"
	"locatorbpla/internal/bot/command/errorreport"
	"locatorbpla/internal/bot/command/settings"
	"locatorbpla/internal/bot/command/start"
	"locatorbpla/internal/bot/input"
	"locatorbpla/internal/bot/router"
	"locatorbpla/internal/bot/screen"
	"locatorbpla/internal/bot/session"
	"locatorbpla/internal/channel"
	"locatorbpla/internal/config"
	"locatorbpla/internal/migrate"
	"locatorbpla/internal/notify"
	"locatorbpla/internal/ops"
	"locatorbpla/internal/repository"
)

const updateWorkers = 1

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)
	cfg, err := config.Load()
	if err != nil {
		log.Error("config", "err", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, cfg, log); err != nil && !errors.Is(err, context.Canceled) {
		log.Error("stopped", "err", err)
		os.Exit(1)
	}
}

func run(sigCtx context.Context, cfg config.Config, log *slog.Logger) error {
	pool, err := repository.Connect(sigCtx, cfg.DSN())
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := migrate.Up(sigCtx, pool); err != nil {
		return err
	}
	if err := repository.Probe(sigCtx, pool); err != nil {
		return err
	}
	owner, err := ops.Hold(sigCtx, pool, log)
	if err != nil {
		return err
	}
	defer owner.Release()
	ctx, cancelWork := context.WithCancel(context.Background())
	defer cancelWork()

	users := repository.NewUsers(pool)
	outbox := notify.NewQueue(users, log)
	hooks := notify.NewHook(log)
	outbox.Out = hooks
	sessions := &session.Service{Repo: repository.NewSessions(pool)}
	ui := &screen.Screen{Sessions: sessions}
	alert := &ops.Alerter{AdminID: cfg.AdminTelegramID, Log: log, Every: 3 * time.Minute}
	go func() {
		select {
		case <-sigCtx.Done():
			cancelWork()
		case <-owner.Lost():
			log.Error("advisory lock lost, stopping bot and reader")
			alert.Infra(context.Background(), errors.New("advisory lock lost"))
			cancelWork()
		}
	}()
	settingsH := &settings.Handler{Users: users, Sessions: sessions, Screen: ui}
	in := &input.Router{
		Users:    users,
		Sessions: sessions,
		Settings: settingsH,
		Screen:   ui,
		AdminID:  cfg.AdminTelegramID,
	}
	clicks := &callback.Router{
		Screen:        ui,
		Settings:      &get.Settings{Settings: settingsH, Screen: ui},
		Locations:     &locations.Handler{Sessions: sessions, Screen: ui},
		Webhook:       &hook.Handler{Users: users, Sessions: sessions, Screen: ui},
		Notifications: &notifications.Handler{Users: users, Settings: settingsH, Screen: ui},
		Cancel:        &cancel.Handler{Users: users, Sessions: sessions, Settings: settingsH, Screen: ui},
		Confirm:       &confirm.Handler{Input: in, Screen: ui},
		Admin:         &admin.Handler{Users: users, Posts: outbox, Screen: ui},
	}
	app := &router.Router{
		Log:      log,
		Screen:   ui,
		Sessions: sessions,
		Start:    &start.Handler{Users: users, Sessions: sessions, Screen: ui},
		Settings: settingsH,
		Report:   &errorreport.Handler{Sessions: sessions, Screen: ui},
		Clicks:   clicks,
		Input:    in,
		Alert:    alert,
	}
	var aborting sync.Map
	deny := func(ctx context.Context, chatID int64) {
		if _, loaded := aborting.LoadOrStore(chatID, struct{}{}); loaded {
			return
		}
		defer aborting.Delete(chatID)
		_ = users.Block(ctx, chatID)
		_ = in.Rollback(ctx, chatID)
	}
	ui.Denied = deny
	outbox.Denied = deny
	tg, err := bot.New(cfg.TelegramToken,
		bot.WithDefaultHandler(app.Handle),
		bot.WithNotAsyncHandlers(),
		bot.WithWorkers(updateWorkers),
		bot.WithAllowedUpdates(bot.AllowedUpdates{"message", "callback_query"}),
		bot.WithErrorsHandler(func(err error) {
			alert.Repeat(context.Background(), err)
		}),
	)
	if err != nil {
		return err
	}
	ui.Bot = tg
	alert.Bot = tg
	in.Bot = tg
	if _, err := tg.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{Command: "start", Description: "Начальное меню"},
			{Command: "settings", Description: "⚙️ Настройки", IsEphemeral: true},
			{Command: "error", Description: "⚠️ Сообщить об ошибке", IsEphemeral: true},
		},
	}); err != nil {
		return fmt.Errorf("set commands: %w", err)
	}
	outbox.Bot = tg
	alertFn := func(err error) { alert.Infra(context.Background(), err) }
	outbox.Alert = alertFn
	mt := repository.NewMTSession(pool)
	sessionData, err := mt.Bootstrap(ctx, cfg.TGSession)
	if err != nil {
		return err
	}
	poller := &channel.Poller{
		APIID:   cfg.APIID,
		APIHash: cfg.APIHash,
		Session: channel.NewStore(sessionData, func(ctx context.Context, data []byte) error {
			return mt.Save(ctx, data, "")
		}),
		Channel: cfg.Channel,
		Every:   cfg.PollInterval,
		Limit:   cfg.KeepAliveFails,
		Subs:    users,
		Queue:   outbox,
		Log:     log,
		Alert:   alertFn,
	}
	keep := &ops.KeepAlive{
		Repo:  repository.NewKeepAlive(pool),
		Every: cfg.KeepAliveEvery,
		Limit: cfg.KeepAliveFails,
		Log:   log,
		Alert: alertFn,
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		tg.Start(ctx)
		return ctx.Err()
	})
	g.Go(func() error { return ops.Serve(ctx, cfg.HTTPAddr, log) })
	g.Go(func() error { return keep.Run(ctx) })
	g.Go(func() error { return poller.Run(ctx) })
	g.Go(func() error { return outbox.Run(ctx) })
	g.Go(func() error { return hooks.Run(ctx) })
	g.Go(func() error { return alert.Run(ctx) })
	return g.Wait()
}

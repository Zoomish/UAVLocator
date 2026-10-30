package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/sync/errgroup"

	"locatorbpla/internal/bot/callback"
	"locatorbpla/internal/bot/callback/admin"
	"locatorbpla/internal/bot/callback/cancel"
	"locatorbpla/internal/bot/callback/get"
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

func run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	pool, err := repository.Connect(ctx, cfg.DSN())
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := migrate.Up(ctx, pool); err != nil {
		return err
	}
	if err := repository.Probe(ctx, pool); err != nil {
		return err
	}

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return err
	}

	users := repository.NewUsers(pool)
	sessions := &session.Service{Repo: repository.NewSessions(pool)}
	ui := &screen.Screen{Bot: bot, Sessions: sessions}
	alert := &ops.Alerter{Bot: bot, AdminID: cfg.AdminTelegramID, Log: log}
	settingsH := &settings.Handler{Users: users, Sessions: sessions, Screen: ui}
	clicks := &callback.Router{
		Screen:        ui,
		Settings:      &get.Settings{Settings: settingsH, Screen: ui},
		Locations:     &locations.Handler{Sessions: sessions, Screen: ui},
		Notifications: &notifications.Handler{Users: users, Settings: settingsH, Screen: ui},
		Cancel:        &cancel.Handler{Sessions: sessions, Settings: settingsH, Screen: ui},
		Admin:         &admin.Handler{Users: users, Screen: ui},
	}
	app := &router.Router{
		Log:      log,
		Bot:      bot,
		Screen:   ui,
		Sessions: sessions,
		Start:    &start.Handler{Users: users, Sessions: sessions, Screen: ui},
		Settings: settingsH,
		Report:   &errorreport.Handler{Sessions: sessions, Screen: ui},
		Clicks:   clicks,
		Input: &input.Router{
			Users:    users,
			Sessions: sessions,
			Settings: settingsH,
			Screen:   ui,
			AdminID:  cfg.AdminTelegramID,
			Bot:      bot,
		},
		Alert: alert,
	}
	sender := &notify.Sender{Bot: bot, Users: users, Log: log}
	alertFn := func(err error) { alert.Infra(context.Background(), err) }
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
		Send:    sender,
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
	g.Go(func() error { return app.Run(ctx) })
	g.Go(func() error { return ops.Serve(ctx, cfg.HTTPAddr, log) })
	g.Go(func() error { return keep.Run(ctx) })
	g.Go(func() error { return poller.Run(ctx) })
	return g.Wait()
}

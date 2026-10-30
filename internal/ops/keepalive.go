package ops

import (
	"context"
	"log/slog"
	"time"

	"locatorbpla/internal/repository"
)

type KeepAlive struct {
	Repo  *repository.KeepAlive
	Every time.Duration
	Limit int
	Log   *slog.Logger
	Alert func(error)
}

func (k *KeepAlive) Run(ctx context.Context) error {
	k.beat(ctx)
	every := k.Every
	if every <= 0 {
		every = 3 * time.Minute
	}
	t := time.NewTicker(every)
	defer t.Stop()
	fails := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if err := k.Repo.Beat(ctx); err != nil {
				fails++
				k.Log.Error("keepalive", "err", err, "fails", fails)
				if k.Alert != nil && k.Limit > 0 && fails%k.Limit == 0 {
					k.Alert(err)
				}
				continue
			}
			fails = 0
			k.Log.Info("keepalive ok")
		}
	}
}

func (k *KeepAlive) beat(ctx context.Context) {
	if err := k.Repo.Beat(ctx); err != nil {
		k.Log.Error("keepalive", "err", err)
		if k.Alert != nil {
			k.Alert(err)
		}
	}
}

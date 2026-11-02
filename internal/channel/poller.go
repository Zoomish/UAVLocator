package channel

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gotd/td/tgerr"

	"locatorbpla/internal/domain"
	"locatorbpla/internal/notify"
)

type Subscribers interface {
	FindNotifiable(ctx context.Context, normalized string) ([]domain.Match, error)
}

type Enqueuer interface {
	Enqueue(ctx context.Context, html string, matches []domain.Match) error
}

type Poller struct {
	APIID   int
	APIHash string
	Session *Store
	Channel string
	Every   time.Duration
	Limit   int
	Subs    Subscribers
	Queue   Enqueuer
	Log     *slog.Logger
	Alert   func(error)
	queued  map[int]struct{}
}

func (p *Poller) Run(ctx context.Context) error {
	if p.APIID == 0 || p.APIHash == "" || p.Channel == "" {
		p.Log.Warn("channel poller disabled")
		<-ctx.Done()
		return ctx.Err()
	}
	every := p.Every
	if every <= 0 {
		every = 2 * time.Minute
	}
	fails := 0
	for {
		err := p.session(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		fails++
		p.Log.Error("channel session", "err", err, "fails", fails)
		if p.Alert != nil && p.Limit > 0 && fails%p.Limit == 0 {
			p.Alert(fmt.Errorf("channel poller: %w", err))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(every):
		}
	}
}

func (p *Poller) session(ctx context.Context) error {
	client, err := New(p.APIID, p.APIHash, p.Session)
	if err != nil {
		return err
	}
	return client.Run(ctx, func(ctx context.Context) error {
		status, err := client.Auth().Status(ctx)
		if err != nil {
			return fmt.Errorf("auth status: %w", err)
		}
		if !status.Authorized {
			return errors.New("reader is not authorized, run cmd/login")
		}
		reader := &Reader{API: client.API(), Channel: p.Channel, Log: p.Log}
		return p.loop(ctx, reader)
	})
}

func (p *Poller) loop(ctx context.Context, reader *Reader) error {
	every := p.Every
	if every <= 0 {
		every = 2 * time.Minute
	}
	if err := p.once(ctx, reader); err != nil && !isFlood(err) {
		p.fail(err)
	}
	t := time.NewTicker(every)
	defer t.Stop()
	fails := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if err := p.once(ctx, reader); err != nil {
				if wait, ok := floodWait(err); ok {
					p.Log.Warn("channel flood wait", "wait", wait.String())
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(wait):
					}
					continue
				}
				fails++
				p.Log.Error("channel poll", "err", err, "fails", fails)
				if p.Alert != nil && p.Limit > 0 && fails%p.Limit == 0 {
					p.Alert(fmt.Errorf("channel poll: %w", err))
				}
				continue
			}
			fails = 0
		}
	}
}

func (p *Poller) once(ctx context.Context, reader *Reader) error {
	batch, err := reader.Unread(ctx)
	if err != nil {
		return err
	}
	if p.queued == nil {
		p.queued = map[int]struct{}{}
	}
	cursor := 0
	for _, item := range batch.Items {
		if item.HTML != "" {
			if _, seen := p.queued[item.ID]; !seen {
				matches, err := p.Subs.FindNotifiable(ctx, domain.Normalize(item.Plain))
				if err != nil {
					_ = reader.MarkRead(ctx, batch.Channel, cursor)
					return fmt.Errorf("match locations: %w", err)
				}
				if len(matches) > 0 {
					if err := p.Queue.Enqueue(ctx, item.HTML, matches); err != nil {
						_ = reader.MarkRead(ctx, batch.Channel, cursor)
						return fmt.Errorf("enqueue delivery: %w", err)
					}
					p.queued[item.ID] = struct{}{}
				}
			}
		}
		cursor = item.ID
		if err := reader.MarkRead(ctx, batch.Channel, cursor); err != nil {
			return err
		}
		delete(p.queued, item.ID)
	}
	return nil
}

func isFlood(err error) bool {
	_, ok := floodWait(err)
	return ok
}

func floodWait(err error) (time.Duration, bool) {
	if wait, ok := tgerr.AsFloodWait(err); ok && wait > 0 {
		return capWait(wait), true
	}
	var retry *notify.RetryAfterError
	if errors.As(err, &retry) && retry != nil && retry.Wait > 0 {
		return capWait(retry.Wait), true
	}
	return 0, false
}

func capWait(wait time.Duration) time.Duration {
	if wait > 5*time.Minute {
		return 5 * time.Minute
	}
	return wait
}

func (p *Poller) fail(err error) {
	p.Log.Error("channel poll", "err", err)
	if p.Alert != nil {
		p.Alert(fmt.Errorf("channel poll: %w", err))
	}
}

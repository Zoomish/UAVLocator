package ops

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const advisoryLockKey int64 = 748001001

type Owner struct {
	pool *pgxpool.Pool
	conn *pgxpool.Conn
	log  *slog.Logger
	lost chan struct{}
	once sync.Once
}

func Hold(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) (*Owner, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("advisory lock connection: %w", err)
	}
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, advisoryLockKey); err != nil {
		conn.Release()
		return nil, fmt.Errorf("advisory lock: %w", err)
	}
	owner := &Owner{
		pool: pool,
		conn: conn,
		log:  log,
		lost: make(chan struct{}),
	}
	go owner.watch(ctx)
	return owner, nil
}

func (o *Owner) Lost() <-chan struct{} {
	return o.lost
}

func (o *Owner) Release() {
	if o == nil || o.conn == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := o.conn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, advisoryLockKey); err != nil {
		o.log.Error("advisory unlock", "err", err)
	}
	o.conn.Release()
	o.conn = nil
}

func (o *Owner) watch(ctx context.Context) {
	tick := time.NewTicker(time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if o.conn == nil {
				return
			}
			if err := o.conn.Ping(ctx); err != nil {
				if ctx.Err() != nil {
					return
				}
				o.log.Error("advisory lock ping", "err", err)
				if err := o.reacquire(ctx); err != nil {
					o.log.Error("advisory lock lost", "err", err)
					o.once.Do(func() { close(o.lost) })
					return
				}
			}
		}
	}
}

func (o *Owner) reacquire(ctx context.Context) error {
	if o.conn != nil {
		o.conn.Release()
		o.conn = nil
	}
	conn, err := o.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("reacquire connection: %w", err)
	}
	var ok bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, advisoryLockKey).Scan(&ok); err != nil {
		conn.Release()
		return fmt.Errorf("try advisory lock: %w", err)
	}
	if !ok {
		conn.Release()
		return errors.New("advisory lock held by another process")
	}
	o.conn = conn
	return nil
}

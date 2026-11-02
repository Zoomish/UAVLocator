package notify

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const hookWorkers = 4

type hookJob struct {
	url  string
	body string
}

type Hook struct {
	Log    *slog.Logger
	ch     chan hookJob
	client *http.Client
}

func NewHook(log *slog.Logger) *Hook {
	return &Hook{
		Log: log,
		ch:  make(chan hookJob, 32),
		client: &http.Client{
			Timeout: 3 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: &http.Transport{
				DisableKeepAlives: true,
				DialContext:       dialPublic,
			},
		},
	}
}

func (h *Hook) Push(url, body string) {
	if h == nil || url == "" {
		return
	}
	select {
	case h.ch <- hookJob{url: url, body: body}:
	default:
		if h.Log != nil {
			h.Log.Warn("webhook dropped", "host", logHost(url))
		}
	}
}

func (h *Hook) Run(ctx context.Context) error {
	for i := 0; i < hookWorkers; i++ {
		go h.loop(ctx)
	}
	<-ctx.Done()
	return nil
}

func (h *Hook) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-h.ch:
			h.post(ctx, job)
		}
	}
}

func (h *Hook) post(ctx context.Context, job hookJob) {
	if err := PublicHTTPS(ctx, job.url); err != nil {
		h.Log.Warn("webhook skipped", "host", logHost(job.url), "err", err)
		return
	}
	payload, err := json.Marshal(map[string]string{"text": job.body})
	if err != nil {
		h.Log.Error("webhook body", "err", err)
		return
	}
	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, job.url, strings.NewReader(string(payload)))
	if err != nil {
		h.Log.Warn("webhook request", "host", logHost(job.url), "err", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil && ctx.Err() == nil {
		h.Log.Warn("webhook post", "host", logHost(job.url), "err", err)
	}
}

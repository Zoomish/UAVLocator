package ops

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

func Listen(addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		return nil, fmt.Errorf("http listen %s: %w", addr, err)
	}
	return ln, nil
}

func Serve(ctx context.Context, ln net.Listener) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/app", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"Hello World!"}`))
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	errCh := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	select {
	case <-ctx.Done():
		shut, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shut); err != nil {
			return fmt.Errorf("http shutdown: %w", err)
		}
		return ctx.Err()
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	}
}

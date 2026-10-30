package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"

	"locatorbpla/internal/channel"
	"locatorbpla/internal/config"
	"locatorbpla/internal/migrate"
	"locatorbpla/internal/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	phone := normalizePhone(os.Getenv("PHONE"))
	if phone == "" {
		fmt.Fprintln(os.Stderr, "PHONE is empty")
		os.Exit(1)
	}
	password := os.Getenv("TG_PASSWORD")
	ctx := context.Background()
	pool, err := repository.Connect(ctx, cfg.DSN())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := migrate.Up(ctx, pool); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	mt := repository.NewMTSession(pool)
	store := channel.NewStore(nil, func(ctx context.Context, data []byte) error {
		return mt.Save(ctx, data, repository.SessionSeed(data))
	})
	client, err := channel.New(cfg.APIID, cfg.APIHash, store)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = client.Run(ctx, func(ctx context.Context) error {
		flow := auth.NewFlow(auth.Constant(phone, password, auth.CodeAuthenticatorFunc(readCode)), auth.SendCodeOptions{})
		if err := client.Auth().IfNecessary(ctx, flow); err != nil {
			return err
		}
		status, err := client.Auth().Status(ctx)
		if err != nil {
			return err
		}
		if !status.Authorized {
			return fmt.Errorf("login finished without authorization")
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	raw := store.Bytes()
	if len(raw) == 0 {
		fmt.Fprintln(os.Stderr, "session is empty")
		os.Exit(1)
	}
	fmt.Println("TG_SESSION=" + repository.EncodeSession(raw))
}

func normalizePhone(phone string) string {
	return strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(strings.TrimSpace(phone))
}

func readCode(ctx context.Context, _ *tg.AuthSentCode) (string, error) {
	if code := strings.TrimSpace(os.Getenv("TG_CODE")); code != "" {
		return code, nil
	}
	path := strings.TrimSpace(os.Getenv("TG_CODE_FILE"))
	if path == "" {
		fmt.Fprint(os.Stderr, "code: ")
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(line), nil
	}
	fmt.Fprintln(os.Stderr, "waiting for Telegram code")
	for {
		buf, err := os.ReadFile(path)
		if err == nil {
			code := strings.TrimSpace(string(buf))
			if code != "" {
				return code, nil
			}
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

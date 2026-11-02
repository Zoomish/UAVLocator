package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken   string
	AdminTelegramID int64
	PGHost          string
	PGPort          string
	PGUser          string
	PGPassword      string
	PGDatabase      string
	APIID           int
	APIHash         string
	TGSession       string
	Channel         string
	PollInterval    time.Duration
	PollFailAlert   int
	HTTPAddr        string
}

func Load() (Config, error) {
	_ = godotenv.Load()
	cfg := Config{
		TelegramToken: strings.TrimSpace(os.Getenv("TELEGRAM_TOKEN")),
		PGHost:        strings.TrimSpace(os.Getenv("POSTGRES_HOST")),
		PGPort:        strings.TrimSpace(os.Getenv("POSTGRES_PORT")),
		PGUser:        strings.TrimSpace(os.Getenv("POSTGRES_USER")),
		PGPassword:    os.Getenv("POSTGRES_PASSWORD"),
		PGDatabase:    strings.TrimSpace(os.Getenv("POSTGRES_DATABASE")),
		APIHash:       strings.TrimSpace(os.Getenv("API_HASH")),
		TGSession:     strings.TrimSpace(os.Getenv("TG_SESSION")),
		Channel:       strings.TrimPrefix(strings.TrimSpace(os.Getenv("CHANNEL")), "@"),
		PollFailAlert: 3,
	}
	if cfg.PGPort == "" {
		cfg.PGPort = "5432"
	}
	cfg.HTTPAddr = ListenAddr(os.Getenv("HTTP_ADDR"), os.Getenv("PORT"))
	var err error
	cfg.PollInterval, err = durationEnv("POLL_INTERVAL", 2*time.Minute)
	if err != nil {
		return Config{}, err
	}
	if v := strings.TrimSpace(os.Getenv("ADMIN_TELEGRAM_ID")); v != "" {
		cfg.AdminTelegramID, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("ADMIN_TELEGRAM_ID: %w", err)
		}
	}
	if v := strings.TrimSpace(os.Getenv("API_ID")); v != "" {
		cfg.APIID, err = strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("API_ID: %w", err)
		}
	}
	if v := strings.TrimSpace(os.Getenv("POLL_FAIL_ALERT")); v != "" {
		cfg.PollFailAlert, err = strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("POLL_FAIL_ALERT: %w", err)
		}
	}
	if cfg.TelegramToken == "" {
		return Config{}, fmt.Errorf("TELEGRAM_TOKEN is empty")
	}
	if cfg.PGHost == "" || cfg.PGUser == "" || cfg.PGDatabase == "" {
		return Config{}, fmt.Errorf("postgres env is incomplete")
	}
	return cfg, nil
}

func (c Config) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.PGUser, c.PGPassword),
		Host:   net.JoinHostPort(c.PGHost, c.PGPort),
		Path:   "/" + c.PGDatabase,
	}
	q := u.Query()
	q.Set("sslmode", sslMode(c.PGHost))
	u.RawQuery = q.Encode()
	return u.String()
}

func ListenAddr(httpAddr, port string) string {
	if p := strings.TrimSpace(port); p != "" {
		return "0.0.0.0:" + p
	}
	addr := strings.TrimSpace(httpAddr)
	if addr == "" {
		return "0.0.0.0:3000"
	}
	if strings.HasPrefix(addr, ":") {
		return "0.0.0.0" + addr
	}
	return addr
}

func sslMode(host string) string {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "localhost", "127.0.0.1", "::1":
		return "disable"
	default:
		return "require"
	}
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return d, nil
}

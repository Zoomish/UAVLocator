package main

import (
	"context"
	"fmt"
	"os"
	"time"

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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
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
	fmt.Println("migrations applied")
}

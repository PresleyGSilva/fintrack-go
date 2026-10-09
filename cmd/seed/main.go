package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/PresleyGSilva/fintrack-go/internal/config"
	"github.com/PresleyGSilva/fintrack-go/internal/db"
	"github.com/PresleyGSilva/fintrack-go/internal/seed"
)

// cmd/seed: popula a primeira org com dados fictícios de demonstração.
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg := config.Load()
	if cfg.DBURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DBURL)
	if err != nil {
		slog.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		slog.Error("migrate", "err", err)
		os.Exit(1)
	}
	var orgID string
	if err := pool.QueryRow(ctx, `SELECT id FROM orgs ORDER BY created_at LIMIT 1`).Scan(&orgID); err != nil {
		slog.Error("no org found (register first)")
		os.Exit(1)
	}
	if err := seed.Run(ctx, pool, orgID); err != nil {
		slog.Error("seed", "err", err)
		os.Exit(1)
	}
	fmt.Println("seed ok")
}

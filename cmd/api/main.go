package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PresleyGSilva/fintrack-go/internal/config"
	"github.com/PresleyGSilva/fintrack-go/internal/db"
	"github.com/PresleyGSilva/fintrack-go/internal/handlers"
	"github.com/PresleyGSilva/fintrack-go/internal/seed"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	if cfg.DBURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return errors.New("JWT_SECRET is required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DBURL)
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if cfg.SeedDemo {
		if err := seedFirstOrg(ctx, pool); err != nil {
			return fmt.Errorf("seed: %w", err)
		}
	}
	h := &handlers.Handler{
		DB:        pool,
		JWTSecret: cfg.JWTSecret,
		Limiter:   handlers.NewLoginLimiter(10, time.Minute),
	}
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h.NewRouter(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shut, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shut)
	}()
	slog.Info("listening", "port", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func seedFirstOrg(ctx context.Context, pool *pgxpool.Pool) error {
	var orgID string
	err := pool.QueryRow(ctx, `SELECT id FROM orgs ORDER BY created_at LIMIT 1`).Scan(&orgID)
	if err != nil {
		slog.Info("seed skipped: no org yet (register first)")
		return nil
	}
	return seed.Run(ctx, pool, orgID)
}

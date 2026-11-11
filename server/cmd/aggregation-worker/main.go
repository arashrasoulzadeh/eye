package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arashrasoulzadeh/eye/internal/config"
	"github.com/arashrasoulzadeh/eye/internal/service"
	"github.com/arashrasoulzadeh/eye/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}
	db, err := storage.OpenDatabase(cfg)
	if err != nil {
		slog.Error("database open failed", "err", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	service.RunAggregationJob(ctx, &service.AggregatorService{DB: db}, time.Minute, slog.Default())
}

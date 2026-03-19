package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/efer92/go-yandex-practicum-metrics/internal/agent"
	"github.com/efer92/go-yandex-practicum-metrics/internal/config"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.ParseAgentConfig(os.Args[1:])
	if err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	ag := agent.NewWithConfig(
		"http://"+cfg.Addr,
		time.Duration(cfg.PollInterval)*time.Second,
		time.Duration(cfg.ReportInterval)*time.Second,
		logger,
		cfg.Key,
		cfg.RateLimit,
	)

	logger.Info("agent started",
		zap.String("server", cfg.Addr),
		zap.Int("poll_interval", cfg.PollInterval),
		zap.Int("report_interval", cfg.ReportInterval),
		zap.Int("rate_limit", cfg.RateLimit),
	)

	ag.Run(ctx)
}

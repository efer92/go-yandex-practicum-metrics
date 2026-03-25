package main

import (
	"log"
	"os"
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
	defer logger.Sync()

	ag := agent.NewWithConfig(
		"http://"+cfg.Addr,
		time.Duration(cfg.PollInterval)*time.Second,
		time.Duration(cfg.ReportInterval)*time.Second,
		logger,
		cfg.Key,
	)

	logger.Info("agent started",
		zap.String("server", cfg.Addr),
		zap.Int("poll_interval", cfg.PollInterval),
		zap.Int("report_interval", cfg.ReportInterval),
	)
	ag.Run()
}

package main

import (
	"log"
	"os"
	"time"

	"github.com/efer92/go-yandex-practicum-metrics/internal/agent"
)

func main() {
	cfg := parseConfig(os.Args[1:])

	serverURL := "http://" + cfg.Addr

	ag := agent.NewWithConfig(
		serverURL,
		time.Duration(cfg.PollInterval)*time.Second,
		time.Duration(cfg.ReportInterval)*time.Second,
	)

	log.Printf("Agent started (server: %s, poll: %ds, report: %ds)",
		cfg.Addr, cfg.PollInterval, cfg.ReportInterval)
	ag.Run()
}

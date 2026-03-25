package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

type AgentConfig struct {
	Addr           string
	PollInterval   int
	ReportInterval int
	Key            string
}

func ParseAgentConfig(args []string) (AgentConfig, error) {
	fs := flag.NewFlagSet("agent", flag.ExitOnError)

	cfg := AgentConfig{}
	fs.StringVar(&cfg.Addr, "a", "localhost:8080", "HTTP server address")
	fs.IntVar(&cfg.ReportInterval, "r", 10, "Report interval in seconds")
	fs.IntVar(&cfg.PollInterval, "p", 2, "Poll interval in seconds")
	fs.StringVar(&cfg.Key, "k", "", "Signing key for HMAC-SHA256")
	fs.Parse(args)

	if env, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Addr = env
	}
	if env, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		v, err := strconv.Atoi(env)
		if err != nil {
			return AgentConfig{}, fmt.Errorf("invalid REPORT_INTERVAL %q: %w", env, err)
		}
		cfg.ReportInterval = v
	}
	if env, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		v, err := strconv.Atoi(env)
		if err != nil {
			return AgentConfig{}, fmt.Errorf("invalid POLL_INTERVAL %q: %w", env, err)
		}
		cfg.PollInterval = v
	}
	if env, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = env
	}

	return cfg, nil
}

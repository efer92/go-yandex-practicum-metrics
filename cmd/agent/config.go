package main

import (
	"flag"
	"os"
	"strconv"
)

type Config struct {
	Addr           string
	PollInterval   int
	ReportInterval int
}

func parseConfig(args []string) Config {
	fs := flag.NewFlagSet("agent", flag.ExitOnError)

	cfg := Config{}
	fs.StringVar(&cfg.Addr, "a", "localhost:8080", "HTTP server address")
	fs.IntVar(&cfg.ReportInterval, "r", 10, "Report interval in seconds")
	fs.IntVar(&cfg.PollInterval, "p", 2, "Poll interval in seconds")
	fs.Parse(args)

	if env, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Addr = env
	}
	if env, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		if v, err := strconv.Atoi(env); err == nil {
			cfg.ReportInterval = v
		}
	}
	if env, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		if v, err := strconv.Atoi(env); err == nil {
			cfg.PollInterval = v
		}
	}

	return cfg
}

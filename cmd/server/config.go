package main

import (
	"flag"
	"os"
)

type Config struct {
	Addr string
}

func parseConfig(args []string) Config {
	fs := flag.NewFlagSet("server", flag.ExitOnError)

	cfg := Config{}
	fs.StringVar(&cfg.Addr, "a", "localhost:8080", "HTTP server address")
	fs.Parse(args)

	if env, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Addr = env
	}

	return cfg
}

package main

import (
	"flag"
	"os"
	"strconv"
)

type Config struct {
	Addr            string
	StoreInterval   int
	FileStoragePath string
	Restore         bool
}

func parseConfig(args []string) Config {
	fs := flag.NewFlagSet("server", flag.ExitOnError)

	cfg := Config{}
	fs.StringVar(&cfg.Addr, "a", "localhost:8080", "HTTP server address")
	fs.IntVar(&cfg.StoreInterval, "i", 300, "Store interval in seconds (0 = sync)")
	fs.StringVar(&cfg.FileStoragePath, "f", "/tmp/metrics-db.json", "File storage path")
	fs.BoolVar(&cfg.Restore, "r", true, "Restore metrics from file on start")
	fs.Parse(args)

	if env, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Addr = env
	}
	if env, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		if v, err := strconv.Atoi(env); err == nil {
			cfg.StoreInterval = v
		}
	}
	if env, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = env
	}
	if env, ok := os.LookupEnv("RESTORE"); ok {
		if v, err := strconv.ParseBool(env); err == nil {
			cfg.Restore = v
		}
	}

	return cfg
}

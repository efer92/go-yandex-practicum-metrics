// Package config parses server and agent configuration from CLI flags and environment variables.
package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

// ServerConfig holds runtime parameters for cmd/server.
type ServerConfig struct {
	Addr            string
	StoreInterval   int
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
	Key             string
	AuditFile       string
	AuditURL        string
}

// ParseServerConfig reads flags from args, then overlays the corresponding environment variables.
func ParseServerConfig(args []string) (ServerConfig, error) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)

	cfg := ServerConfig{}
	fs.StringVar(&cfg.Addr, "a", "localhost:8080", "HTTP server address")
	fs.IntVar(&cfg.StoreInterval, "i", 300, "Store interval in seconds (0 = sync)")
	fs.StringVar(&cfg.FileStoragePath, "f", "/tmp/metrics-db.json", "File storage path")
	fs.BoolVar(&cfg.Restore, "r", true, "Restore metrics from file on start")
	fs.StringVar(&cfg.DatabaseDSN, "d", "", "PostgreSQL DSN")
	fs.StringVar(&cfg.Key, "k", "", "Signing key for HMAC-SHA256")
	fs.StringVar(&cfg.AuditFile, "audit-file", "", "Audit log file path (audit disabled if empty)")
	fs.StringVar(&cfg.AuditURL, "audit-url", "", "Audit log HTTP observer URL (audit disabled if empty)")
	fs.Parse(args)

	if env, ok := os.LookupEnv("ADDRESS"); ok {
		cfg.Addr = env
	}
	if env, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		v, err := strconv.Atoi(env)
		if err != nil {
			return ServerConfig{}, fmt.Errorf("invalid STORE_INTERVAL %q: %w", env, err)
		}
		cfg.StoreInterval = v
	}
	if env, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = env
	}
	if env, ok := os.LookupEnv("RESTORE"); ok {
		v, err := strconv.ParseBool(env)
		if err != nil {
			return ServerConfig{}, fmt.Errorf("invalid RESTORE %q: %w", env, err)
		}
		cfg.Restore = v
	}
	if env, ok := os.LookupEnv("DATABASE_DSN"); ok {
		cfg.DatabaseDSN = env
	}
	if env, ok := os.LookupEnv("KEY"); ok {
		cfg.Key = env
	}
	if env, ok := os.LookupEnv("AUDIT_FILE"); ok {
		cfg.AuditFile = env
	}
	if env, ok := os.LookupEnv("AUDIT_URL"); ok {
		cfg.AuditURL = env
	}

	return cfg, nil
}

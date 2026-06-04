// Package config parses server and agent configuration from CLI flags,
// environment variables and an optional JSON config file. Precedence
// is: env > flag > config file > hardcoded default.
package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
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
	CryptoKey       string
}

// ParseServerConfig reads CLI flags, environment variables and an optional
// JSON config file (selected via -c/-config/CONFIG). Higher precedence wins.
func ParseServerConfig(args []string) (ServerConfig, error) {
	addrDefault := "localhost:8080"
	storeIntervalDefault := 300
	fileStorageDefault := "/tmp/metrics-db.json"
	restoreDefault := true
	dsnDefault := ""
	cryptoKeyDefault := ""

	if path := extractConfigPath(args); path != "" {
		fc, err := loadServerFile(path)
		if err != nil {
			return ServerConfig{}, err
		}
		if fc.Address != nil {
			addrDefault = *fc.Address
		}
		if fc.Restore != nil {
			restoreDefault = *fc.Restore
		}
		if fc.StoreInterval != nil {
			d, err := time.ParseDuration(*fc.StoreInterval)
			if err != nil {
				return ServerConfig{}, fmt.Errorf("config %q: invalid store_interval %q: %w", path, *fc.StoreInterval, err)
			}
			storeIntervalDefault = int(d.Seconds())
		}
		if fc.StoreFile != nil {
			fileStorageDefault = *fc.StoreFile
		}
		if fc.DatabaseDSN != nil {
			dsnDefault = *fc.DatabaseDSN
		}
		if fc.CryptoKey != nil {
			cryptoKeyDefault = *fc.CryptoKey
		}
	}

	fs := flag.NewFlagSet("server", flag.ExitOnError)

	cfg := ServerConfig{}
	fs.StringVar(&cfg.Addr, "a", addrDefault, "HTTP server address")
	fs.IntVar(&cfg.StoreInterval, "i", storeIntervalDefault, "Store interval in seconds (0 = sync)")
	fs.StringVar(&cfg.FileStoragePath, "f", fileStorageDefault, "File storage path")
	fs.BoolVar(&cfg.Restore, "r", restoreDefault, "Restore metrics from file on start")
	fs.StringVar(&cfg.DatabaseDSN, "d", dsnDefault, "PostgreSQL DSN")
	fs.StringVar(&cfg.Key, "k", "", "Signing key for HMAC-SHA256")
	fs.StringVar(&cfg.AuditFile, "audit-file", "", "Audit log file path (audit disabled if empty)")
	fs.StringVar(&cfg.AuditURL, "audit-url", "", "Audit log HTTP observer URL (audit disabled if empty)")
	fs.StringVar(&cfg.CryptoKey, "crypto-key", cryptoKeyDefault, "Path to PEM-encoded RSA private key for decrypting agent payloads")
	var configPathFlag string
	fs.StringVar(&configPathFlag, "c", "", "Path to JSON config file (overridden by CONFIG env)")
	fs.StringVar(&configPathFlag, "config", "", "Path to JSON config file (overridden by CONFIG env)")
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
	if env, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		cfg.CryptoKey = env
	}

	return cfg, nil
}

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Pointer fields distinguish "absent in JSON" (nil) from "present and zero".
type serverFile struct {
	Address       *string `json:"address,omitempty"`
	Restore       *bool   `json:"restore,omitempty"`
	StoreInterval *string `json:"store_interval,omitempty"`
	StoreFile     *string `json:"store_file,omitempty"`
	DatabaseDSN   *string `json:"database_dsn,omitempty"`
	CryptoKey     *string `json:"crypto_key,omitempty"`
	TrustedSubnet *string `json:"trusted_subnet,omitempty"`
}

type agentFile struct {
	Address        *string `json:"address,omitempty"`
	ReportInterval *string `json:"report_interval,omitempty"`
	PollInterval   *string `json:"poll_interval,omitempty"`
	CryptoKey      *string `json:"crypto_key,omitempty"`
}

func loadServerFile(path string) (*serverFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	var f serverFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	return &f, nil
}

func loadAgentFile(path string) (*agentFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	var f agentFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	return &f, nil
}

// extractConfigPath returns the JSON config path from CONFIG env or from
// -c / -config / --config arg (with optional =value form). CONFIG env wins.
func extractConfigPath(args []string) string {
	if env, ok := os.LookupEnv("CONFIG"); ok && env != "" {
		return env
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-c", a == "-config", a == "--config":
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(a, "-c="):
			return strings.TrimPrefix(a, "-c=")
		case strings.HasPrefix(a, "-config="):
			return strings.TrimPrefix(a, "-config=")
		case strings.HasPrefix(a, "--config="):
			return strings.TrimPrefix(a, "--config=")
		}
	}
	return ""
}

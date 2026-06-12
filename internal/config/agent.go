package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

// AgentConfig holds runtime parameters for cmd/agent.
type AgentConfig struct {
	Addr           string
	PollInterval   int
	ReportInterval int
	Key            string
	RateLimit      int
	CryptoKey      string
	GRPCAddr       string
}

// ParseAgentConfig reads CLI flags, environment variables and an optional
// JSON config file (selected via -c/-config/CONFIG). Higher precedence wins.
func ParseAgentConfig(args []string) (AgentConfig, error) {
	addrDefault := "localhost:8080"
	pollDefault := 2
	reportDefault := 10
	cryptoKeyDefault := ""
	grpcAddrDefault := ""

	if path := extractConfigPath(args); path != "" {
		fc, err := loadAgentFile(path)
		if err != nil {
			return AgentConfig{}, err
		}
		if fc.Address != nil {
			addrDefault = *fc.Address
		}
		if fc.PollInterval != nil {
			d, err := time.ParseDuration(*fc.PollInterval)
			if err != nil {
				return AgentConfig{}, fmt.Errorf("config %q: invalid poll_interval %q: %w", path, *fc.PollInterval, err)
			}
			pollDefault = int(d.Seconds())
		}
		if fc.ReportInterval != nil {
			d, err := time.ParseDuration(*fc.ReportInterval)
			if err != nil {
				return AgentConfig{}, fmt.Errorf("config %q: invalid report_interval %q: %w", path, *fc.ReportInterval, err)
			}
			reportDefault = int(d.Seconds())
		}
		if fc.CryptoKey != nil {
			cryptoKeyDefault = *fc.CryptoKey
		}
		if fc.GRPCAddress != nil {
			grpcAddrDefault = *fc.GRPCAddress
		}
	}

	fs := flag.NewFlagSet("agent", flag.ExitOnError)

	cfg := AgentConfig{}
	fs.StringVar(&cfg.Addr, "a", addrDefault, "HTTP server address")
	fs.IntVar(&cfg.ReportInterval, "r", reportDefault, "Report interval in seconds")
	fs.IntVar(&cfg.PollInterval, "p", pollDefault, "Poll interval in seconds")
	fs.StringVar(&cfg.Key, "k", "", "Signing key for HMAC-SHA256")
	fs.IntVar(&cfg.RateLimit, "l", 1, "Max concurrent outgoing requests")
	fs.StringVar(&cfg.CryptoKey, "crypto-key", cryptoKeyDefault, "Path to PEM-encoded RSA public key for encrypting outgoing payloads")
	fs.StringVar(&cfg.GRPCAddr, "grpc-address", grpcAddrDefault, "gRPC server address (empty = use HTTP transport)")
	// configPathFlag is registered solely so fs.Parse does not error on
	// the -c / -config user-facing flag — the actual value has already been
	// consumed by extractConfigPath above.
	var configPathFlag string
	fs.StringVar(&configPathFlag, "c", "", "Path to JSON config file (overridden by CONFIG env)")
	fs.StringVar(&configPathFlag, "config", "", "Path to JSON config file (overridden by CONFIG env)")
	fs.Parse(args)
	_ = configPathFlag

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
	if env, ok := os.LookupEnv("RATE_LIMIT"); ok {
		v, err := strconv.Atoi(env)
		if err != nil {
			return AgentConfig{}, fmt.Errorf("invalid RATE_LIMIT %q: %w", env, err)
		}
		cfg.RateLimit = v
	}
	if env, ok := os.LookupEnv("CRYPTO_KEY"); ok {
		cfg.CryptoKey = env
	}
	if env, ok := os.LookupEnv("GRPC_ADDRESS"); ok {
		cfg.GRPCAddr = env
	}

	return cfg, nil
}

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

func TestParseServerConfig_FromFile(t *testing.T) {
	path := writeFile(t, `{
        "address": "file:9000",
        "restore": false,
        "store_interval": "30s",
        "store_file": "/var/file.db",
        "database_dsn": "dsn://from-file",
        "crypto_key": "/keys/priv.pem",
        "trusted_subnet": "10.0.0.0/8"
    }`)

	cfg, err := ParseServerConfig([]string{"-c", path})
	require.NoError(t, err)

	assert.Equal(t, "file:9000", cfg.Addr)
	assert.False(t, cfg.Restore)
	assert.Equal(t, 30, cfg.StoreInterval)
	assert.Equal(t, "/var/file.db", cfg.FileStoragePath)
	assert.Equal(t, "dsn://from-file", cfg.DatabaseDSN)
	assert.Equal(t, "/keys/priv.pem", cfg.CryptoKey)
	assert.Equal(t, "10.0.0.0/8", cfg.TrustedSubnet)
}

func TestParseConfig_GRPCAddress(t *testing.T) {
	// file < flag < env, same as every other option
	srvPath := writeFile(t, `{"grpc_address": "file:3200"}`)
	cfg, err := ParseServerConfig([]string{"-c", srvPath})
	require.NoError(t, err)
	assert.Equal(t, "file:3200", cfg.GRPCAddr)

	cfg, err = ParseServerConfig([]string{"-c", srvPath, "-grpc-address", "flag:3201"})
	require.NoError(t, err)
	assert.Equal(t, "flag:3201", cfg.GRPCAddr)

	t.Setenv("GRPC_ADDRESS", "env:3202")
	cfg, err = ParseServerConfig([]string{"-grpc-address", "flag:3201"})
	require.NoError(t, err)
	assert.Equal(t, "env:3202", cfg.GRPCAddr)

	acfg, err := ParseAgentConfig([]string{})
	require.NoError(t, err)
	assert.Equal(t, "env:3202", acfg.GRPCAddr)
}

func TestParseServerConfig_TrustedSubnetFlagAndEnv(t *testing.T) {
	cfg, err := ParseServerConfig([]string{"-t", "192.168.0.0/16"})
	require.NoError(t, err)
	assert.Equal(t, "192.168.0.0/16", cfg.TrustedSubnet)

	t.Setenv("TRUSTED_SUBNET", "172.16.0.0/12")
	cfg, err = ParseServerConfig([]string{"-t", "192.168.0.0/16"})
	require.NoError(t, err)
	assert.Equal(t, "172.16.0.0/12", cfg.TrustedSubnet)
}

func TestParseServerConfig_FlagOverridesFile(t *testing.T) {
	path := writeFile(t, `{"address": "file:9000", "store_interval": "30s"}`)
	cfg, err := ParseServerConfig([]string{"-c", path, "-a", "flag:1234", "-i", "5"})
	require.NoError(t, err)
	assert.Equal(t, "flag:1234", cfg.Addr)
	assert.Equal(t, 5, cfg.StoreInterval)
}

func TestParseServerConfig_EnvOverridesFile(t *testing.T) {
	path := writeFile(t, `{"address": "file:9000", "restore": false}`)
	t.Setenv("ADDRESS", "env:7777")
	t.Setenv("RESTORE", "true")
	cfg, err := ParseServerConfig([]string{"-c", path})
	require.NoError(t, err)
	assert.Equal(t, "env:7777", cfg.Addr)
	assert.True(t, cfg.Restore)
}

func TestParseServerConfig_ConfigEnvSelectsFile(t *testing.T) {
	path := writeFile(t, `{"address": "via-env:5555"}`)
	t.Setenv("CONFIG", path)
	cfg, err := ParseServerConfig([]string{})
	require.NoError(t, err)
	assert.Equal(t, "via-env:5555", cfg.Addr)
}

func TestParseServerConfig_BadFileReturnsError(t *testing.T) {
	_, err := ParseServerConfig([]string{"-c", "/nonexistent/path/to/config.json"})
	assert.Error(t, err)
}

func TestParseServerConfig_BadStoreIntervalReturnsError(t *testing.T) {
	path := writeFile(t, `{"store_interval": "not-a-duration"}`)
	_, err := ParseServerConfig([]string{"-c", path})
	assert.Error(t, err)
}

func TestParseAgentConfig_FromFile(t *testing.T) {
	path := writeFile(t, `{
        "address": "file:9000",
        "report_interval": "15s",
        "poll_interval": "3s",
        "crypto_key": "/keys/pub.pem"
    }`)
	cfg, err := ParseAgentConfig([]string{"-c", path})
	require.NoError(t, err)
	assert.Equal(t, "file:9000", cfg.Addr)
	assert.Equal(t, 15, cfg.ReportInterval)
	assert.Equal(t, 3, cfg.PollInterval)
	assert.Equal(t, "/keys/pub.pem", cfg.CryptoKey)
}

func TestParseAgentConfig_FlagOverridesFile(t *testing.T) {
	path := writeFile(t, `{"poll_interval": "5s", "report_interval": "20s"}`)
	cfg, err := ParseAgentConfig([]string{"-c", path, "-p", "1", "-r", "2"})
	require.NoError(t, err)
	assert.Equal(t, 1, cfg.PollInterval)
	assert.Equal(t, 2, cfg.ReportInterval)
}

func TestParseAgentConfig_EnvOverridesFile(t *testing.T) {
	path := writeFile(t, `{"poll_interval": "5s"}`)
	t.Setenv("POLL_INTERVAL", "9")
	cfg, err := ParseAgentConfig([]string{"-c", path})
	require.NoError(t, err)
	assert.Equal(t, 9, cfg.PollInterval)
}

func TestExtractConfigPath(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"dash-c value", []string{"-c", "a.json"}, "a.json"},
		{"dash-c equals", []string{"-c=b.json"}, "b.json"},
		{"dash-config equals", []string{"-config=c.json"}, "c.json"},
		{"double-dash equals", []string{"--config=d.json"}, "d.json"},
		{"double-dash space", []string{"--config", "e.json"}, "e.json"},
		{"no flag", []string{"-a", "localhost:8080"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// extractConfigPath treats an empty CONFIG as "unset";
			// t.Setenv also schedules a clean restore at test end.
			t.Setenv("CONFIG", "")
			assert.Equal(t, tc.want, extractConfigPath(tc.args))
		})
	}
}

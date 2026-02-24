package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseServerConfig_Defaults(t *testing.T) {
	cfg, err := ParseServerConfig([]string{})
	require.NoError(t, err)

	assert.Equal(t, "localhost:8080", cfg.Addr)
	assert.Equal(t, 300, cfg.StoreInterval)
	assert.Equal(t, "/tmp/metrics-db.json", cfg.FileStoragePath)
	assert.Equal(t, true, cfg.Restore)
}

func TestParseServerConfig_Flags(t *testing.T) {
	cfg, err := ParseServerConfig([]string{"-a", "localhost:9090", "-i", "60", "-f", "/tmp/test.json", "-r=false"})
	require.NoError(t, err)

	assert.Equal(t, "localhost:9090", cfg.Addr)
	assert.Equal(t, 60, cfg.StoreInterval)
	assert.Equal(t, "/tmp/test.json", cfg.FileStoragePath)
	assert.Equal(t, false, cfg.Restore)
}

func TestParseServerConfig_EnvOverridesFlags(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:7070")
	t.Setenv("STORE_INTERVAL", "10")
	t.Setenv("FILE_STORAGE_PATH", "/tmp/env.json")
	t.Setenv("RESTORE", "false")

	cfg, err := ParseServerConfig([]string{"-a", "localhost:9090", "-i", "300", "-f", "/tmp/flag.json", "-r=true"})
	require.NoError(t, err)

	assert.Equal(t, "localhost:7070", cfg.Addr)
	assert.Equal(t, 10, cfg.StoreInterval)
	assert.Equal(t, "/tmp/env.json", cfg.FileStoragePath)
	assert.Equal(t, false, cfg.Restore)
}

func TestParseServerConfig_StoreIntervalZero(t *testing.T) {
	cfg, err := ParseServerConfig([]string{"-i", "0"})
	require.NoError(t, err)
	assert.Equal(t, 0, cfg.StoreInterval)
}

func TestParseServerConfig_InvalidStoreInterval(t *testing.T) {
	t.Setenv("STORE_INTERVAL", "not-a-number")

	_, err := ParseServerConfig([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "STORE_INTERVAL")
}

func TestParseServerConfig_InvalidRestore(t *testing.T) {
	t.Setenv("RESTORE", "maybe")

	_, err := ParseServerConfig([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RESTORE")
}

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerParseConfig_Defaults(t *testing.T) {
	cfg := parseConfig([]string{})

	assert.Equal(t, "localhost:8080", cfg.Addr)
	assert.Equal(t, 300, cfg.StoreInterval)
	assert.Equal(t, "/tmp/metrics-db.json", cfg.FileStoragePath)
	assert.Equal(t, true, cfg.Restore)
}

func TestServerParseConfig_Flags(t *testing.T) {
	cfg := parseConfig([]string{"-a", "localhost:9090", "-i", "60", "-f", "/tmp/test.json", "-r=false"})

	assert.Equal(t, "localhost:9090", cfg.Addr)
	assert.Equal(t, 60, cfg.StoreInterval)
	assert.Equal(t, "/tmp/test.json", cfg.FileStoragePath)
	assert.Equal(t, false, cfg.Restore)
}

func TestServerParseConfig_EnvOverridesFlags(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:7070")
	t.Setenv("STORE_INTERVAL", "10")
	t.Setenv("FILE_STORAGE_PATH", "/tmp/env.json")
	t.Setenv("RESTORE", "false")

	cfg := parseConfig([]string{"-a", "localhost:9090", "-i", "300", "-f", "/tmp/flag.json", "-r=true"})

	assert.Equal(t, "localhost:7070", cfg.Addr)
	assert.Equal(t, 10, cfg.StoreInterval)
	assert.Equal(t, "/tmp/env.json", cfg.FileStoragePath)
	assert.Equal(t, false, cfg.Restore)
}

func TestServerParseConfig_StoreIntervalZero(t *testing.T) {
	cfg := parseConfig([]string{"-i", "0"})
	assert.Equal(t, 0, cfg.StoreInterval)
}

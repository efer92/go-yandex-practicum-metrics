package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentParseConfig_Defaults(t *testing.T) {
	cfg := parseConfig([]string{})

	assert.Equal(t, "localhost:8080", cfg.Addr)
	assert.Equal(t, 2, cfg.PollInterval)
	assert.Equal(t, 10, cfg.ReportInterval)
}

func TestAgentParseConfig_Flags(t *testing.T) {
	cfg := parseConfig([]string{"-a", "localhost:9090", "-p", "5", "-r", "15"})

	assert.Equal(t, "localhost:9090", cfg.Addr)
	assert.Equal(t, 5, cfg.PollInterval)
	assert.Equal(t, 15, cfg.ReportInterval)
}

func TestAgentParseConfig_EnvOverridesDefaults(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:7070")
	t.Setenv("POLL_INTERVAL", "3")
	t.Setenv("REPORT_INTERVAL", "20")

	cfg := parseConfig([]string{})

	assert.Equal(t, "localhost:7070", cfg.Addr)
	assert.Equal(t, 3, cfg.PollInterval)
	assert.Equal(t, 20, cfg.ReportInterval)
}

func TestAgentParseConfig_EnvOverridesFlags(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:7070")
	t.Setenv("POLL_INTERVAL", "3")

	cfg := parseConfig([]string{"-a", "localhost:9090", "-p", "10", "-r", "20"})

	assert.Equal(t, "localhost:7070", cfg.Addr) // env побеждает
	assert.Equal(t, 3, cfg.PollInterval)        // env побеждает
	assert.Equal(t, 20, cfg.ReportInterval)     // флаг, т.к. env не задан
}

func TestAgentParseConfig_InvalidEnvInterval(t *testing.T) {
	t.Setenv("POLL_INTERVAL", "not-a-number")
	t.Setenv("REPORT_INTERVAL", "abc")

	cfg := parseConfig([]string{"-p", "5", "-r", "15"})

	// невалидный env игнорируется, остаётся значение флага
	assert.Equal(t, 5, cfg.PollInterval)
	assert.Equal(t, 15, cfg.ReportInterval)
}

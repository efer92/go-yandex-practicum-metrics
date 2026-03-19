package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAgentConfig_Defaults(t *testing.T) {
	cfg, err := ParseAgentConfig([]string{})
	require.NoError(t, err)

	assert.Equal(t, "localhost:8080", cfg.Addr)
	assert.Equal(t, 2, cfg.PollInterval)
	assert.Equal(t, 10, cfg.ReportInterval)
	assert.Equal(t, "", cfg.Key)
	assert.Equal(t, 1, cfg.RateLimit)
}

func TestParseAgentConfig_Flags(t *testing.T) {
	cfg, err := ParseAgentConfig([]string{"-a", "localhost:9090", "-p", "5", "-r", "15", "-l", "3"})
	require.NoError(t, err)

	assert.Equal(t, "localhost:9090", cfg.Addr)
	assert.Equal(t, 5, cfg.PollInterval)
	assert.Equal(t, 15, cfg.ReportInterval)
	assert.Equal(t, 3, cfg.RateLimit)
}

func TestParseAgentConfig_EnvOverridesDefaults(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:7070")
	t.Setenv("POLL_INTERVAL", "3")
	t.Setenv("REPORT_INTERVAL", "20")
	t.Setenv("RATE_LIMIT", "5")

	cfg, err := ParseAgentConfig([]string{})
	require.NoError(t, err)

	assert.Equal(t, "localhost:7070", cfg.Addr)
	assert.Equal(t, 3, cfg.PollInterval)
	assert.Equal(t, 20, cfg.ReportInterval)
	assert.Equal(t, 5, cfg.RateLimit)
}

func TestParseAgentConfig_EnvOverridesFlags(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:7070")
	t.Setenv("POLL_INTERVAL", "3")
	t.Setenv("RATE_LIMIT", "10")

	cfg, err := ParseAgentConfig([]string{"-a", "localhost:9090", "-p", "10", "-r", "20", "-l", "2"})
	require.NoError(t, err)

	assert.Equal(t, "localhost:7070", cfg.Addr)
	assert.Equal(t, 3, cfg.PollInterval)
	assert.Equal(t, 20, cfg.ReportInterval)
	assert.Equal(t, 10, cfg.RateLimit)
}

func TestParseAgentConfig_InvalidPollInterval(t *testing.T) {
	t.Setenv("POLL_INTERVAL", "not-a-number")

	_, err := ParseAgentConfig([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "POLL_INTERVAL")
}

func TestParseAgentConfig_InvalidReportInterval(t *testing.T) {
	t.Setenv("REPORT_INTERVAL", "abc")

	_, err := ParseAgentConfig([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "REPORT_INTERVAL")
}

func TestParseAgentConfig_InvalidRateLimit(t *testing.T) {
	t.Setenv("RATE_LIMIT", "not-a-number")

	_, err := ParseAgentConfig([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "RATE_LIMIT")
}

func TestParseAgentConfig_KeyFlag(t *testing.T) {
	cfg, err := ParseAgentConfig([]string{"-k", "mysecret"})
	require.NoError(t, err)
	assert.Equal(t, "mysecret", cfg.Key)
}

func TestParseAgentConfig_KeyEnv(t *testing.T) {
	t.Setenv("KEY", "envsecret")

	cfg, err := ParseAgentConfig([]string{})
	require.NoError(t, err)
	assert.Equal(t, "envsecret", cfg.Key)
}

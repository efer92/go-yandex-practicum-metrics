package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerParseConfig_Defaults(t *testing.T) {
	cfg := parseConfig([]string{})

	assert.Equal(t, "localhost:8080", cfg.Addr)
}

func TestServerParseConfig_Flag(t *testing.T) {
	cfg := parseConfig([]string{"-a", "localhost:9090"})

	assert.Equal(t, "localhost:9090", cfg.Addr)
}

func TestServerParseConfig_EnvOverridesDefault(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:7070")

	cfg := parseConfig([]string{})

	assert.Equal(t, "localhost:7070", cfg.Addr)
}

func TestServerParseConfig_EnvOverridesFlag(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:7070")

	cfg := parseConfig([]string{"-a", "localhost:9090"})

	assert.Equal(t, "localhost:7070", cfg.Addr)
}

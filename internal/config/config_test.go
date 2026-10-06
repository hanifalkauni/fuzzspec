package config_test

import (
	"path/filepath"
	"testing"

	"github.com/fuzzspec/fuzzspec/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_ExampleFile(t *testing.T) {
	configPath, err := filepath.Abs("../../fuzzspec.example.yaml")
	require.NoError(t, err)

	cfg, err := config.LoadConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "http://localhost:8000", cfg.Target)
	assert.Equal(t, "./api/openapi.yaml", cfg.Spec)
	assert.True(t, cfg.AutoDiscover)
	assert.Equal(t, 15, cfg.Execution.Concurrency)
	assert.Equal(t, 50, cfg.Execution.RPS)
	assert.True(t, cfg.Execution.SafeMode)
	assert.Equal(t, "gemini", cfg.AI.Provider)
	assert.True(t, cfg.Oracles.FailOn5xx)
	assert.True(t, cfg.Reporting.SanitizePII)

	// Check custom languages
	require.Len(t, cfg.CustomLanguages, 1)
	assert.Equal(t, "elixir-phoenix", cfg.CustomLanguages[0].Name)
	assert.Contains(t, cfg.CustomLanguages[0].DiscoveryEndpoints, "/api/openapi")
}

func TestLoadConfig_Default(t *testing.T) {
	cfg, err := config.LoadConfig("")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, 10, cfg.Execution.Concurrency)
	assert.Equal(t, 20, cfg.Execution.RPS)
	assert.True(t, cfg.Execution.SafeMode)
}

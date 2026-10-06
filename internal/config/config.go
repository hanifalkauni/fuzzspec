package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the declarative configuration for FuzzSpec.
type Config struct {
	Version        string          `yaml:"version"`
	Target         string          `yaml:"target"`
	Spec           string          `yaml:"spec"`
	AutoDiscover   bool            `yaml:"auto_discover"`
	Execution      ExecutionConfig `yaml:"execution"`
	Authentication AuthConfig      `yaml:"authentication"`
	Filtering      FilterConfig    `yaml:"filtering"`
	AI             AIConfig        `yaml:"ai"`
	Oracles        OracleConfig    `yaml:"oracles"`
	CustomLanguages []CustomLang   `yaml:"custom_languages"`
	Reporting      ReportingConfig `yaml:"reporting"`
}

type ExecutionConfig struct {
	Concurrency int    `yaml:"concurrency"`
	RPS         int    `yaml:"rps"`
	Timeout     string `yaml:"timeout"`
	Retries     int    `yaml:"retries"`
	SafeMode    bool   `yaml:"safe_mode"`
}

type AuthConfig struct {
	Type     string            `yaml:"type"` // "bearer", "apikey", "custom"
	TokenEnv string            `yaml:"token_env"`
	Headers  map[string]string `yaml:"headers"`
}

type FilterConfig struct {
	IncludePaths []string `yaml:"include_paths"`
	ExcludePaths []string `yaml:"exclude_paths"`
	Methods      []string `yaml:"methods"`
}

type AIConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Provider     string `yaml:"provider"` // "gemini", "openai", "anthropic", "local"
	Model        string `yaml:"model"`
	CacheVectors bool   `yaml:"cache_vectors"`
	CacheDir     string `yaml:"cache_dir"`
}

type OracleConfig struct {
	FailOn5xx          bool `yaml:"fail_on_5xx"`
	FailOnSchemaDrift   bool `yaml:"fail_on_schema_drift"`
	ScanInfoLeak       bool `yaml:"scan_info_leak"`
	LatencyThresholdMs int  `yaml:"latency_threshold_ms"`
}

type CustomLang struct {
	Name                       string   `yaml:"name"`
	DisplayName                string   `yaml:"display_name"`
	DiscoveryEndpoints         []string `yaml:"discovery_endpoints"`
	StacktracePatterns         []string `yaml:"stacktrace_patterns"`
	ValidationErrorSignatures  []string `yaml:"validation_error_signatures"`
}

type ReportingConfig struct {
	Terminal    bool   `yaml:"terminal"`
	Markdown    string `yaml:"markdown"`
	SARIF       string `yaml:"sarif"`
	JUnit       string `yaml:"junit"`
	JSON        string `yaml:"json"`
	SanitizePII bool   `yaml:"sanitize_pii"`
}

// DefaultConfig returns a configuration struct populated with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Version:      "1",
		AutoDiscover: false,
		Execution: ExecutionConfig{
			Concurrency: 10,
			RPS:         20,
			Timeout:     "5s",
			Retries:     2,
			SafeMode:    true,
		},
		AI: AIConfig{
			Enabled:      true,
			Provider:     "gemini",
			Model:        "gemini-1.5-flash",
			CacheVectors: true,
			CacheDir:     ".fuzzspec-cache",
		},
		Oracles: OracleConfig{
			FailOn5xx:          true,
			FailOnSchemaDrift:   true,
			ScanInfoLeak:       true,
			LatencyThresholdMs: 3000,
		},
		Reporting: ReportingConfig{
			Terminal:    true,
			SanitizePII: true,
		},
	}
}

// LoadConfig loads and parses a YAML configuration file from disk.
func LoadConfig(filePath string) (*Config, error) {
	cfg := DefaultConfig()

	if filePath == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file '%s': %w", filePath, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML configuration: %w", err)
	}

	return cfg, nil
}

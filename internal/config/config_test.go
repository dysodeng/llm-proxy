package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string // empty string means no file (use a path that does not exist)
		wantErr     bool
		check       func(t *testing.T, cfg *Config)
	}{
		{
			name: "valid config file",
			yaml: `
server:
  port: 9090

log:
  level: debug
  file: /tmp/proxy.log
  max_age: 7

rate_limit:
  enabled: false
  default:
    requests_per_second: 5
    burst: 10
  whitelist:
    - "sk-abc"
    - "sk-def"
  overrides:
    "sk-override":
      requests_per_second: 50
      burst: 100

providers:
  openai:
    base_url: "https://custom.openai.com"
  anthropic:
    base_url: "https://custom.anthropic.com"
`,
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				t.Helper()

				// Server
				if cfg.Server.Port != 9090 {
					t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
				}

				// Log
				if cfg.Log.Level != "debug" {
					t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "debug")
				}
				if cfg.Log.File != "/tmp/proxy.log" {
					t.Errorf("Log.File = %q, want %q", cfg.Log.File, "/tmp/proxy.log")
				}
				if cfg.Log.MaxAge != 7 {
					t.Errorf("Log.MaxAge = %d, want 7", cfg.Log.MaxAge)
				}

				// RateLimit
				if cfg.RateLimit.Enabled {
					t.Errorf("RateLimit.Enabled = true, want false")
				}
				if cfg.RateLimit.Default.RequestsPerSecond != 5 {
					t.Errorf("RateLimit.Default.RequestsPerSecond = %f, want 5", cfg.RateLimit.Default.RequestsPerSecond)
				}
				if cfg.RateLimit.Default.Burst != 10 {
					t.Errorf("RateLimit.Default.Burst = %d, want 10", cfg.RateLimit.Default.Burst)
				}
				if len(cfg.RateLimit.Whitelist) != 2 {
					t.Errorf("RateLimit.Whitelist length = %d, want 2", len(cfg.RateLimit.Whitelist))
				} else {
					if cfg.RateLimit.Whitelist[0] != "sk-abc" {
						t.Errorf("RateLimit.Whitelist[0] = %q, want %q", cfg.RateLimit.Whitelist[0], "sk-abc")
					}
					if cfg.RateLimit.Whitelist[1] != "sk-def" {
						t.Errorf("RateLimit.Whitelist[1] = %q, want %q", cfg.RateLimit.Whitelist[1], "sk-def")
					}
				}
				if len(cfg.RateLimit.Overrides) != 1 {
					t.Errorf("RateLimit.Overrides length = %d, want 1", len(cfg.RateLimit.Overrides))
				} else {
					override, ok := cfg.RateLimit.Overrides["sk-override"]
					if !ok {
						t.Error("RateLimit.Overrides missing key \"sk-override\"")
					} else {
						if override.RequestsPerSecond != 50 {
							t.Errorf("override.RequestsPerSecond = %f, want 50", override.RequestsPerSecond)
						}
						if override.Burst != 100 {
							t.Errorf("override.Burst = %d, want 100", override.Burst)
						}
					}
				}

				// Providers
				if cfg.Providers.OpenAI.BaseURL != "https://custom.openai.com" {
					t.Errorf("Providers.OpenAI.BaseURL = %q, want %q", cfg.Providers.OpenAI.BaseURL, "https://custom.openai.com")
				}
				if cfg.Providers.Anthropic.BaseURL != "https://custom.anthropic.com" {
					t.Errorf("Providers.Anthropic.BaseURL = %q, want %q", cfg.Providers.Anthropic.BaseURL, "https://custom.anthropic.com")
				}
			},
		},
		{
			name:    "missing config file uses defaults",
			yaml:    "", // no file written; a non-existent path will be used
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				t.Helper()

				// Defaults
				if cfg.Server.Port != 8080 {
					t.Errorf("Server.Port = %d, want 8080 (default)", cfg.Server.Port)
				}
				if cfg.Log.Level != "info" {
					t.Errorf("Log.Level = %q, want %q (default)", cfg.Log.Level, "info")
				}
				if cfg.Log.MaxAge != 30 {
					t.Errorf("Log.MaxAge = %d, want 30 (default)", cfg.Log.MaxAge)
				}
				if !cfg.RateLimit.Enabled {
					t.Error("RateLimit.Enabled = false, want true (default)")
				}
				if cfg.RateLimit.Default.RequestsPerSecond != 10 {
					t.Errorf("RateLimit.Default.RequestsPerSecond = %f, want 10 (default)", cfg.RateLimit.Default.RequestsPerSecond)
				}
				if cfg.RateLimit.Default.Burst != 20 {
					t.Errorf("RateLimit.Default.Burst = %d, want 20 (default)", cfg.RateLimit.Default.Burst)
				}
				if cfg.Providers.OpenAI.BaseURL != "https://api.openai.com" {
					t.Errorf("Providers.OpenAI.BaseURL = %q, want %q (default)", cfg.Providers.OpenAI.BaseURL, "https://api.openai.com")
				}
				if cfg.Providers.Anthropic.BaseURL != "https://api.anthropic.com" {
					t.Errorf("Providers.Anthropic.BaseURL = %q, want %q (default)", cfg.Providers.Anthropic.BaseURL, "https://api.anthropic.com")
				}
			},
		},
		{
			name: "partial config merges with defaults",
			yaml: `
server:
  port: 7070
`,
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				t.Helper()

				// Overridden value
				if cfg.Server.Port != 7070 {
					t.Errorf("Server.Port = %d, want 7070", cfg.Server.Port)
				}
				// Remaining defaults
				if cfg.Log.Level != "info" {
					t.Errorf("Log.Level = %q, want %q (default)", cfg.Log.Level, "info")
				}
				if cfg.RateLimit.Default.RequestsPerSecond != 10 {
					t.Errorf("RateLimit.Default.RequestsPerSecond = %f, want 10 (default)", cfg.RateLimit.Default.RequestsPerSecond)
				}
				if cfg.Providers.OpenAI.BaseURL != "https://api.openai.com" {
					t.Errorf("Providers.OpenAI.BaseURL = %q, want %q (default)", cfg.Providers.OpenAI.BaseURL, "https://api.openai.com")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var configPath string

			if tc.yaml != "" {
				// Write to a temp file.
				f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
				if err != nil {
					t.Fatalf("failed to create temp config file: %v", err)
				}
				if _, err := f.WriteString(tc.yaml); err != nil {
					t.Fatalf("failed to write temp config file: %v", err)
				}
				if err := f.Close(); err != nil {
					t.Fatalf("failed to close temp config file: %v", err)
				}
				configPath = f.Name()
			} else {
				// Use a path that does not exist.
				configPath = t.TempDir() + "/nonexistent-config.yaml"
			}

			cfg, err := Load(configPath)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected an error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg == nil {
				t.Fatal("cfg is nil")
			}

			tc.check(t, cfg)
		})
	}
}

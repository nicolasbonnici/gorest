package config

import "testing"

// A production config that passes every check, for the cases below to vary one
// field at a time from.
func hardenedConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:             3000,
			Environment:      "production",
			Scheme:           "https",
			CORSOrigins:      "https://app.example.com",
			RateLimitEnabled: true,
		},
		Database:   DatabaseConfig{URL: "postgres://localhost/db"},
		Pagination: PaginationConfig{DefaultLimit: 10, MaxLimit: 100},
		Codegen: CodegenConfig{
			Output: OutputConfig{Models: "models", Resources: "resources", DTOs: "dtos"},
		},
	}
}

func TestProductionHardening_Accepts(t *testing.T) {
	if err := hardenedConfig().Validate(); err != nil {
		t.Fatalf("a fully hardened production config was rejected: %v", err)
	}
}

func TestProductionHardening_Rejects(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{"wildcard CORS", func(c *Config) { c.Server.CORSOrigins = "*" }},
		{"wildcard CORS among others", func(c *Config) { c.Server.CORSOrigins = "https://app.example.com, *" }},
		{"rate limiting off", func(c *Config) { c.Server.RateLimitEnabled = false }},
		{"plain http", func(c *Config) { c.Server.Scheme = "http" }},
		{"placeholder jwt secret", func(c *Config) {
			c.Auth.Enabled = true
			c.Auth.JWTSecret = "your-secret-key-minimum-32-characters-long"
		}},
		{"changeme jwt secret", func(c *Config) {
			c.Auth.Enabled = true
			c.Auth.JWTSecret = "changeme-changeme-changeme-changeme-1234"
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := hardenedConfig()
			tt.mutate(cfg)
			if err := cfg.Validate(); err == nil {
				t.Error("production config was accepted but should have been rejected")
			}
		})
	}
}

// The same settings are ordinary on a developer's machine and must not block a
// local run.
func TestHardeningDoesNotApplyOutsideProduction(t *testing.T) {
	for _, env := range []string{"development", "staging", "test", ""} {
		cfg := hardenedConfig()
		cfg.Server.Environment = env
		cfg.Server.Scheme = "http"
		cfg.Server.CORSOrigins = "*"
		cfg.Server.RateLimitEnabled = false
		if err := cfg.Validate(); err != nil {
			t.Errorf("environment %q was rejected: %v", env, err)
		}
	}
}

// The check keys off the environment name, which operators capitalise freely.
func TestHardeningIsCaseInsensitive(t *testing.T) {
	for _, env := range []string{"Production", "PRODUCTION"} {
		cfg := hardenedConfig()
		cfg.Server.Environment = env
		cfg.Server.RateLimitEnabled = false
		if err := cfg.Validate(); err == nil {
			t.Errorf("environment %q skipped the hardening checks", env)
		}
	}
}

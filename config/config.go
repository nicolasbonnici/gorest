package config

import (
	"fmt"
	"strings"
)

// Config represents the complete GoREST configuration
type Config struct {
	Generate   GenerateConfig   `yaml:"generate"`
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Auth       AuthConfig       `yaml:"auth"`
	Pagination PaginationConfig `yaml:"pagination"`
	CORS       CORSConfig       `yaml:"cors"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
}

// GenerateConfig contains code generation settings
type GenerateConfig struct {
	Output OutputConfig `yaml:"output"`
	Auth   GenAuthConfig `yaml:"auth"`
}

// OutputConfig specifies where generated code should be placed
type OutputConfig struct {
	Models    string `yaml:"models"`
	Resources string `yaml:"resources"`
	DTOs      string `yaml:"dtos"`
	OpenAPI   string `yaml:"openapi"`
	Config    string `yaml:"config"`
}

// GenAuthConfig contains auth generation defaults
type GenAuthConfig struct {
	Enabled   bool                `yaml:"enabled"`
	Endpoints AuthEndpointsConfig `yaml:"endpoints"`
}

// AuthEndpointsConfig specifies which endpoints require auth by default
type AuthEndpointsConfig struct {
	List   bool `yaml:"list"`
	Get    bool `yaml:"get"`
	Create bool `yaml:"create"`
	Update bool `yaml:"update"`
	Delete bool `yaml:"delete"`
}

// ServerConfig contains server runtime settings
type ServerConfig struct {
	Port        int    `yaml:"port"`
	Environment string `yaml:"environment"`
}

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	URL string `yaml:"url"`
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	JWT JWTConfig `yaml:"jwt"`
}

// JWTConfig contains JWT-specific settings
type JWTConfig struct {
	Secret string `yaml:"secret"`
	TTL    int    `yaml:"ttl"` // in seconds
}

// PaginationConfig contains pagination settings
type PaginationConfig struct {
	DefaultLimit int `yaml:"default_limit"`
	MaxLimit     int `yaml:"max_limit"`
}

// CORSConfig contains CORS settings
type CORSConfig struct {
	Origins interface{} `yaml:"origins"` // can be string "*" or []string
}

// RateLimitConfig contains rate limiting settings
type RateLimitConfig struct {
	Enabled            bool `yaml:"enabled"`
	RequestsPerSecond int  `yaml:"requests_per_second"`
	Burst              int  `yaml:"burst"`
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate database URL
	if c.Database.URL == "" {
		return fmt.Errorf("database.url is required")
	}

	// Validate JWT secret
	if c.Auth.JWT.Secret == "" {
		return fmt.Errorf("auth.jwt.secret is required")
	}

	// Validate JWT secret length (minimum 32 characters recommended)
	if len(c.Auth.JWT.Secret) < 32 {
		return fmt.Errorf("auth.jwt.secret must be at least 32 characters long for security")
	}

	// Validate JWT TTL
	if c.Auth.JWT.TTL <= 0 {
		return fmt.Errorf("auth.jwt.ttl must be positive")
	}

	// Validate pagination
	if c.Pagination.DefaultLimit <= 0 {
		return fmt.Errorf("pagination.default_limit must be positive")
	}
	if c.Pagination.MaxLimit <= 0 {
		return fmt.Errorf("pagination.max_limit must be positive")
	}
	if c.Pagination.DefaultLimit > c.Pagination.MaxLimit {
		return fmt.Errorf("pagination.default_limit cannot exceed max_limit")
	}

	// Validate server port
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}

	// Validate rate limit settings if enabled
	if c.RateLimit.Enabled {
		if c.RateLimit.RequestsPerSecond <= 0 {
			return fmt.Errorf("rate_limit.requests_per_second must be positive when rate limiting is enabled")
		}
		if c.RateLimit.Burst <= 0 {
			return fmt.Errorf("rate_limit.burst must be positive when rate limiting is enabled")
		}
	}

	// Validate output paths
	if c.Generate.Output.Models == "" {
		return fmt.Errorf("generate.output.models is required")
	}
	if c.Generate.Output.Resources == "" {
		return fmt.Errorf("generate.output.resources is required")
	}
	if c.Generate.Output.DTOs == "" {
		return fmt.Errorf("generate.output.dtos is required")
	}

	return nil
}

// GetCORSOrigins returns CORS origins as a slice of strings
func (c *Config) GetCORSOrigins() []string {
	switch v := c.CORS.Origins.(type) {
	case string:
		if v == "*" {
			return []string{"*"}
		}
		return []string{v}
	case []interface{}:
		origins := make([]string, len(v))
		for i, origin := range v {
			origins[i] = fmt.Sprint(origin)
		}
		return origins
	case []string:
		return v
	default:
		return []string{"*"}
	}
}

// WarnProductionSettings warns about insecure settings in production
func (c *Config) WarnProductionSettings() []string {
	var warnings []string

	if c.Server.Environment == "production" {
		// Check for wildcard CORS
		origins := c.GetCORSOrigins()
		if len(origins) == 1 && origins[0] == "*" {
			warnings = append(warnings, "CORS is set to '*' (allow all) in production - this is insecure")
		}

		// Check for SSL in database URL
		if !strings.Contains(c.Database.URL, "sslmode=require") && !strings.Contains(c.Database.URL, "sslmode=verify-full") {
			if strings.Contains(c.Database.URL, "postgres://") {
				warnings = append(warnings, "Database connection does not use SSL (sslmode=require or sslmode=verify-full recommended)")
			}
		}

		// Check if secret is in YAML (not environment variable)
		if !strings.HasPrefix(c.Auth.JWT.Secret, "${") && len(c.Auth.JWT.Secret) < 64 {
			warnings = append(warnings, "JWT secret appears to be directly in config file - use environment variables for secrets")
		}
	}

	return warnings
}

// SetDefaults sets default values for optional fields
func (c *Config) SetDefaults() {
	// Server defaults
	if c.Server.Port == 0 {
		c.Server.Port = 3000
	}
	if c.Server.Environment == "" {
		c.Server.Environment = "development"
	}

	// Auth defaults
	if c.Auth.JWT.TTL == 0 {
		c.Auth.JWT.TTL = 900 // 15 minutes
	}

	// Pagination defaults
	if c.Pagination.DefaultLimit == 0 {
		c.Pagination.DefaultLimit = 10
	}
	if c.Pagination.MaxLimit == 0 {
		c.Pagination.MaxLimit = 1000
	}

	// CORS defaults
	if c.CORS.Origins == nil {
		c.CORS.Origins = "*"
	}

	// Rate limit defaults
	if c.RateLimit.RequestsPerSecond == 0 {
		c.RateLimit.RequestsPerSecond = 100
	}
	if c.RateLimit.Burst == 0 {
		c.RateLimit.Burst = 200
	}

	// Generate defaults
	if c.Generate.Output.Models == "" {
		c.Generate.Output.Models = "generated/models"
	}
	if c.Generate.Output.Resources == "" {
		c.Generate.Output.Resources = "generated/resources"
	}
	if c.Generate.Output.DTOs == "" {
		c.Generate.Output.DTOs = "generated/dtos"
	}
	if c.Generate.Output.OpenAPI == "" {
		c.Generate.Output.OpenAPI = "generated/openapi"
	}
	if c.Generate.Output.Config == "" {
		c.Generate.Output.Config = "generated/config"
	}
}

package config

import (
	"fmt"
	"strings"
)

type Config struct {
	Generate   GenerateConfig   `yaml:"generate"`
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Auth       AuthConfig       `yaml:"auth"`
	Pagination PaginationConfig `yaml:"pagination"`
	CORS       CORSConfig       `yaml:"cors"`
	RateLimit  RateLimitConfig  `yaml:"rate_limit"`
	Plugins    PluginsConfig    `yaml:"plugins"`
}

type PluginsConfig struct {
	Global []PluginConfig `yaml:"global"`
	Route  []PluginConfig `yaml:"route"`
}

type PluginConfig struct {
	Name    string                 `yaml:"name"`
	Enabled bool                   `yaml:"enabled"`
	Config  map[string]interface{} `yaml:"config"`
}

type GenerateConfig struct {
	Output OutputConfig  `yaml:"output"`
	Auth   GenAuthConfig `yaml:"auth"`
}

type OutputConfig struct {
	Models    string `yaml:"models"`
	Resources string `yaml:"resources"`
	DTOs      string `yaml:"dtos"`
	OpenAPI   string `yaml:"openapi"`
	Config    string `yaml:"config"`
}

type GenAuthConfig struct {
	Enabled   bool                `yaml:"enabled"`
	Endpoints AuthEndpointsConfig `yaml:"endpoints"`
}

type AuthEndpointsConfig struct {
	List   bool `yaml:"list"`
	Get    bool `yaml:"get"`
	Create bool `yaml:"create"`
	Update bool `yaml:"update"`
	Delete bool `yaml:"delete"`
}

type ServerConfig struct {
	Port        int    `yaml:"port"`
	Environment string `yaml:"environment"`
}

type DatabaseConfig struct {
	URL string `yaml:"url"`
}

type AuthConfig struct {
	JWT JWTConfig `yaml:"jwt"`
}

type JWTConfig struct {
	Secret string `yaml:"secret"`
	TTL    int    `yaml:"ttl"`
}

type PaginationConfig struct {
	DefaultLimit int `yaml:"default_limit"`
	MaxLimit     int `yaml:"max_limit"`
}

type CORSConfig struct {
	Origins interface{} `yaml:"origins"`
}

type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerSecond int  `yaml:"requests_per_second"`
	Burst             int  `yaml:"burst"`
}

func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("database.url is required")
	}

	if c.Auth.JWT.Secret == "" {
		return fmt.Errorf("auth.jwt.secret is required")
	}

	if len(c.Auth.JWT.Secret) < 32 {
		return fmt.Errorf("auth.jwt.secret must be at least 32 characters long for security")
	}

	if c.Auth.JWT.TTL <= 0 {
		return fmt.Errorf("auth.jwt.ttl must be positive")
	}

	if c.Pagination.DefaultLimit <= 0 {
		return fmt.Errorf("pagination.default_limit must be positive")
	}
	if c.Pagination.MaxLimit <= 0 {
		return fmt.Errorf("pagination.max_limit must be positive")
	}
	if c.Pagination.DefaultLimit > c.Pagination.MaxLimit {
		return fmt.Errorf("pagination.default_limit cannot exceed max_limit")
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}

	if c.RateLimit.Enabled {
		if c.RateLimit.RequestsPerSecond <= 0 {
			return fmt.Errorf("rate_limit.requests_per_second must be positive when rate limiting is enabled")
		}
		if c.RateLimit.Burst <= 0 {
			return fmt.Errorf("rate_limit.burst must be positive when rate limiting is enabled")
		}
	}

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

func (c *Config) WarnProductionSettings() []string {
	var warnings []string

	if c.Server.Environment == "production" {
		origins := c.GetCORSOrigins()
		if len(origins) == 1 && origins[0] == "*" {
			warnings = append(warnings, "CORS is set to '*' (allow all) in production - this is insecure")
		}

		if !strings.Contains(c.Database.URL, "sslmode=require") && !strings.Contains(c.Database.URL, "sslmode=verify-full") {
			if strings.Contains(c.Database.URL, "postgres://") {
				warnings = append(warnings, "Database connection does not use SSL (sslmode=require or sslmode=verify-full recommended)")
			}
		}

		if !strings.HasPrefix(c.Auth.JWT.Secret, "${") && len(c.Auth.JWT.Secret) < 64 {
			warnings = append(warnings, "JWT secret appears to be directly in config file - use environment variables for secrets")
		}
	}

	return warnings
}

func (c *Config) SetDefaults() {
	if c.Server.Port == 0 {
		c.Server.Port = 3000
	}
	if c.Server.Environment == "" {
		c.Server.Environment = "development"
	}

	if c.Auth.JWT.TTL == 0 {
		c.Auth.JWT.TTL = 900 // 15 minutes
	}

	if c.Pagination.DefaultLimit == 0 {
		c.Pagination.DefaultLimit = 10
	}
	if c.Pagination.MaxLimit == 0 {
		c.Pagination.MaxLimit = 1000
	}

	if c.CORS.Origins == nil {
		c.CORS.Origins = "*"
	}

	if c.RateLimit.RequestsPerSecond == 0 {
		c.RateLimit.RequestsPerSecond = 100
	}
	if c.RateLimit.Burst == 0 {
		c.RateLimit.Burst = 200
	}

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

	// Populate plugins config if not specified (for backward compatibility)
	c.PopulatePluginsFromLegacyConfig()
}

// PopulatePluginsFromLegacyConfig creates plugin configurations from legacy config values
// This ensures backward compatibility with existing gorest.yaml files
func (c *Config) PopulatePluginsFromLegacyConfig() {
	// If plugins are already configured, don't override
	if len(c.Plugins.Global) > 0 || len(c.Plugins.Route) > 0 {
		return
	}

	// Create default plugin configuration
	c.Plugins.Global = []PluginConfig{
		{Name: "requestid", Enabled: true, Config: make(map[string]interface{})},
		{Name: "ratelimit", Enabled: c.RateLimit.Enabled, Config: map[string]interface{}{
			"requests_per_second": c.RateLimit.RequestsPerSecond,
			"burst":               c.RateLimit.Burst,
		}},
		{Name: "cors", Enabled: true, Config: map[string]interface{}{
			"origins": c.GetCORSOriginsString(),
		}},
		{Name: "security", Enabled: true, Config: make(map[string]interface{})},
		{Name: "contenttype", Enabled: true, Config: make(map[string]interface{})},
		{Name: "logger", Enabled: true, Config: make(map[string]interface{})},
	}

	c.Plugins.Route = []PluginConfig{
		{Name: "auth", Enabled: c.Generate.Auth.Enabled, Config: map[string]interface{}{
			"jwt_secret": c.Auth.JWT.Secret,
		}},
	}
}

// GetCORSOriginsString returns CORS origins as a string for plugin configuration
func (c *Config) GetCORSOriginsString() string {
	origins := c.GetCORSOrigins()
	if len(origins) == 1 {
		return origins[0]
	}
	return strings.Join(origins, ",")
}

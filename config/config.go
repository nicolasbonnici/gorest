package config

import (
	"fmt"
	"strings"
)

type Config struct {
	Codegen    CodegenConfig    `yaml:"codegen"`
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Pagination PaginationConfig `yaml:"pagination"`
	Plugins    PluginsConfig    `yaml:"plugins"`
}

type PluginsConfig []PluginConfig

type PluginConfig struct {
	Name    string                 `yaml:"name"`
	Enabled bool                   `yaml:"enabled"`
	Config  map[string]interface{} `yaml:"config"`
}

type CodegenConfig struct {
	Output OutputConfig      `yaml:"output"`
	Auth   CodegenAuthConfig `yaml:"auth"`
}

type OutputConfig struct {
	Models    string `yaml:"models"`
	Resources string `yaml:"resources"`
	DTOs      string `yaml:"dtos"`
	OpenAPI   string `yaml:"openapi"`
	Config    string `yaml:"config"`
}

type CodegenAuthConfig struct {
	Enabled   bool                 `yaml:"enabled"`
	Defaults  map[string]bool      `yaml:"defaults"`
	Endpoints []EndpointAuthConfig `yaml:"endpoints"`
}

type EndpointAuthConfig struct {
	Name   string `yaml:"name"`
	GET    *bool  `yaml:"GET,omitempty"`
	POST   *bool  `yaml:"POST,omitempty"`
	PUT    *bool  `yaml:"PUT,omitempty"`
	DELETE *bool  `yaml:"DELETE,omitempty"`
	PATCH  *bool  `yaml:"PATCH,omitempty"`
}

type ServerConfig struct {
	Scheme             string `yaml:"scheme"`
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	Environment        string `yaml:"environment"`
	CORSOrigins        string `yaml:"cors_origins"`
	RateLimitRPS       int    `yaml:"ratelimit_requests_per_second"`
	RateLimitBurst     int    `yaml:"ratelimit_burst"`
	RateLimitEnabled   bool   `yaml:"ratelimit_enabled"`
	CompressionEnabled bool   `yaml:"compression_enabled"`
	CompressionLevel   int    `yaml:"compression_level"`
}

type DatabaseConfig struct {
	URL string `yaml:"url"`
}

type PaginationConfig struct {
	DefaultLimit int `yaml:"default_limit"`
	MaxLimit     int `yaml:"max_limit"`
}

func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("database.url is required")
	}

	for _, plugin := range c.Plugins {
		if plugin.Name == "auth" && plugin.Enabled {
			if jwtSecret, ok := plugin.Config["jwt_secret"].(string); ok {
				if jwtSecret == "" {
					return fmt.Errorf("plugins.auth.config.jwt_secret is required when auth plugin is enabled")
				}
				if len(jwtSecret) < 32 {
					return fmt.Errorf("plugins.auth.config.jwt_secret must be at least 32 characters long for security")
				}
			} else {
				return fmt.Errorf("plugins.auth.config.jwt_secret is required when auth plugin is enabled")
			}
		}
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

	if c.Codegen.Output.Models == "" {
		return fmt.Errorf("codegen.output.models is required")
	}
	if c.Codegen.Output.Resources == "" {
		return fmt.Errorf("codegen.output.resources is required")
	}
	if c.Codegen.Output.DTOs == "" {
		return fmt.Errorf("codegen.output.dtos is required")
	}

	// Validate codegen.auth.defaults if specified
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true,
	}

	for method := range c.Codegen.Auth.Defaults {
		upperMethod := strings.ToUpper(method)
		if !validMethods[upperMethod] {
			return fmt.Errorf("codegen.auth.defaults: unsupported HTTP method '%s' (supported: GET, POST, PUT, DELETE, PATCH)", method)
		}
		// Normalize method to uppercase
		if method != upperMethod {
			c.Codegen.Auth.Defaults[upperMethod] = c.Codegen.Auth.Defaults[method]
			delete(c.Codegen.Auth.Defaults, method)
		}
	}

	// Validate codegen.auth.endpoints configuration
	endpointNames := make(map[string]bool)

	for i, endpoint := range c.Codegen.Auth.Endpoints {
		// Normalize name to lowercase
		c.Codegen.Auth.Endpoints[i].Name = strings.ToLower(endpoint.Name)
		normalizedName := c.Codegen.Auth.Endpoints[i].Name

		if normalizedName == "" {
			return fmt.Errorf("codegen.auth.endpoints[%d]: name cannot be empty", i)
		}

		// Check for duplicates
		if endpointNames[normalizedName] {
			return fmt.Errorf("duplicate endpoint name: %s", normalizedName)
		}
		endpointNames[normalizedName] = true

		// Note: HTTP method fields (GET, POST, PUT, DELETE, PATCH) are pointers
		// and are validated by the YAML unmarshaler as booleans
	}

	return nil
}

func (c *Config) SetDefaults() {
	if c.Server.Scheme == "" {
		c.Server.Scheme = "http"
	}
	if c.Server.Host == "" {
		c.Server.Host = "localhost"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8000
	}
	if c.Server.Environment == "" {
		c.Server.Environment = "development"
	}
	if c.Server.RateLimitRPS == 0 {
		c.Server.RateLimitRPS = 100
	}
	if c.Server.RateLimitBurst == 0 {
		c.Server.RateLimitBurst = 200
	}
	// CompressionEnabled defaults to true if not explicitly set
	// CompressionLevel defaults to 2 (balanced) if not set or invalid
	if c.Server.CompressionLevel == 0 {
		c.Server.CompressionLevel = 2
	}

	if c.Pagination.DefaultLimit == 0 {
		c.Pagination.DefaultLimit = 10
	}
	if c.Pagination.MaxLimit == 0 {
		c.Pagination.MaxLimit = 1000
	}

	if c.Codegen.Output.Models == "" {
		c.Codegen.Output.Models = "generated/models"
	}
	if c.Codegen.Output.Resources == "" {
		c.Codegen.Output.Resources = "generated/resources"
	}
	if c.Codegen.Output.DTOs == "" {
		c.Codegen.Output.DTOs = "generated/dtos"
	}
	if c.Codegen.Output.OpenAPI == "" {
		c.Codegen.Output.OpenAPI = "generated/openapi"
	}
	if c.Codegen.Output.Config == "" {
		c.Codegen.Output.Config = "generated/config"
	}
}

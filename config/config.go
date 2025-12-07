package config

import (
	"fmt"
	"strings"
)

type Config struct {
	Generate     GenerateConfig   `yaml:"generate"`
	Server       ServerConfig     `yaml:"server"`
	Database     DatabaseConfig   `yaml:"database"`
	Pagination   PaginationConfig `yaml:"pagination"`
	Plugins      PluginsConfig    `yaml:"plugins"`
	Resources    []ResourceConfig `yaml:"resources"`
	AuthDefaults map[string]bool  `yaml:"auth_defaults"` // New: top-level auth defaults
}

type PluginsConfig []PluginConfig

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
	Default   DefaultAuthConfig   `yaml:"default"`
	Endpoints AuthEndpointsConfig `yaml:"endpoints"` // Legacy support
}

type DefaultAuthConfig struct {
	Methods map[string]bool `yaml:"methods"`
}

type AuthEndpointsConfig struct {
	List   bool `yaml:"list"`
	Get    bool `yaml:"get"`
	Create bool `yaml:"create"`
	Update bool `yaml:"update"`
	Delete bool `yaml:"delete"`
}

type ResourceConfig struct {
	Name string          `yaml:"name"`
	Auth map[string]bool `yaml:"auth"` // Flattened: auth: { GET: true, POST: false }
}

type ServerConfig struct {
	Port        int    `yaml:"port"`
	Environment string `yaml:"environment"`
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

	if c.Generate.Output.Models == "" {
		return fmt.Errorf("generate.output.models is required")
	}
	if c.Generate.Output.Resources == "" {
		return fmt.Errorf("generate.output.resources is required")
	}
	if c.Generate.Output.DTOs == "" {
		return fmt.Errorf("generate.output.dtos is required")
	}

	// Validate resources configuration
	resourceNames := make(map[string]bool)
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true,
	}

	for i, resource := range c.Resources {
		// Normalize name to lowercase
		c.Resources[i].Name = strings.ToLower(resource.Name)
		normalizedName := c.Resources[i].Name

		if normalizedName == "" {
			return fmt.Errorf("resource[%d]: name cannot be empty", i)
		}

		// Check for duplicates
		if resourceNames[normalizedName] {
			return fmt.Errorf("duplicate resource name: %s", normalizedName)
		}
		resourceNames[normalizedName] = true

		// Validate and normalize HTTP methods in flattened format
		for method := range resource.Auth {
			upperMethod := strings.ToUpper(method)
			if !validMethods[upperMethod] {
				return fmt.Errorf("resource '%s': unsupported HTTP method '%s' (supported: GET, POST, PUT, DELETE, PATCH)",
					normalizedName, method)
			}
			// Normalize method to uppercase
			if method != upperMethod {
				c.Resources[i].Auth[upperMethod] = resource.Auth[method]
				delete(c.Resources[i].Auth, method)
			}
		}
	}

	// Validate auth_defaults if specified
	for method := range c.AuthDefaults {
		upperMethod := strings.ToUpper(method)
		if !validMethods[upperMethod] {
			return fmt.Errorf("auth_defaults: unsupported HTTP method '%s' (supported: GET, POST, PUT, DELETE, PATCH)", method)
		}
		// Normalize method to uppercase
		if method != upperMethod {
			c.AuthDefaults[upperMethod] = c.AuthDefaults[method]
			delete(c.AuthDefaults, method)
		}
	}

	return nil
}

func (c *Config) SetDefaults() {
	if c.Server.Port == 0 {
		c.Server.Port = 3000
	}
	if c.Server.Environment == "" {
		c.Server.Environment = "development"
	}

	if c.Pagination.DefaultLimit == 0 {
		c.Pagination.DefaultLimit = 10
	}
	if c.Pagination.MaxLimit == 0 {
		c.Pagination.MaxLimit = 1000
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
}

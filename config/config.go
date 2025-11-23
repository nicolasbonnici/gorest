package config

import "fmt"

type Config struct {
	Generate   GenerateConfig   `yaml:"generate"`
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Pagination PaginationConfig `yaml:"pagination"`
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

type PaginationConfig struct {
	DefaultLimit int `yaml:"default_limit"`
	MaxLimit     int `yaml:"max_limit"`
}

func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("database.url is required")
	}

	for _, plugin := range c.Plugins.Route {
		if plugin.Name == "auth" && plugin.Enabled {
			if jwtSecret, ok := plugin.Config["jwt_secret"].(string); ok {
				if jwtSecret == "" {
					return fmt.Errorf("plugins.route.auth.config.jwt_secret is required when auth plugin is enabled")
				}
				if len(jwtSecret) < 32 {
					return fmt.Errorf("plugins.route.auth.config.jwt_secret must be at least 32 characters long for security")
				}
			} else {
				return fmt.Errorf("plugins.route.auth.config.jwt_secret is required when auth plugin is enabled")
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

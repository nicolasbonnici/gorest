package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var envVarRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

func Load(configPath string) (*Config, error) {
	baseConfigFile := filepath.Join(configPath, "gorest.yaml")
	if _, err := os.Stat(baseConfigFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("gorest.yaml not found in %s - configuration file is required", configPath)
	}

	baseConfig, err := loadConfigFile(baseConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load base config: %w", err)
	}

	environment := baseConfig.Server.Environment
	if environment == "" {
		environment = os.Getenv("ENVIRONMENT")
	}
	if environment == "" {
		environment = "development"
	}

	envConfigFile := filepath.Join(configPath, fmt.Sprintf("gorest.%s.yaml", environment))
	if _, err := os.Stat(envConfigFile); err == nil {
		envConfig, err := loadConfigFile(envConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load environment config: %w", err)
		}
		baseConfig = mergeConfigs(baseConfig, envConfig)
	}

	baseConfig.SetDefaults()

	if err := interpolateEnvVars(baseConfig); err != nil {
		return nil, fmt.Errorf("failed to interpolate environment variables: %w", err)
	}

	if err := baseConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return baseConfig, nil
}

func loadConfigFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}

func interpolateEnvVars(config *Config) error {
	config.Database.URL = interpolateString(config.Database.URL)

	for i := range config.Plugins.Global {
		interpolatePluginConfig(config.Plugins.Global[i].Config)
	}

	for i := range config.Plugins.Route {
		interpolatePluginConfig(config.Plugins.Route[i].Config)
	}

	if strings.HasPrefix(config.Database.URL, "${") && strings.HasSuffix(config.Database.URL, "}") {
		varName := strings.TrimSuffix(strings.TrimPrefix(config.Database.URL, "${"), "}")
		return fmt.Errorf("environment variable %s not found (required for database.url)", varName)
	}

	return nil
}

func interpolatePluginConfig(cfg map[string]interface{}) {
	for key, value := range cfg {
		if strVal, ok := value.(string); ok {
			cfg[key] = interpolateString(strVal)
		}
	}
}

func interpolateString(s string) string {
	return envVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[2 : len(match)-1]
		value := os.Getenv(varName)
		if value != "" {
			return value
		}
		return match
	})
}

func mergeConfigs(base, override *Config) *Config {
	result := *base

	if override.Generate.Output.Models != "" {
		result.Generate.Output.Models = override.Generate.Output.Models
	}
	if override.Generate.Output.Resources != "" {
		result.Generate.Output.Resources = override.Generate.Output.Resources
	}
	if override.Generate.Output.DTOs != "" {
		result.Generate.Output.DTOs = override.Generate.Output.DTOs
	}
	if override.Generate.Output.OpenAPI != "" {
		result.Generate.Output.OpenAPI = override.Generate.Output.OpenAPI
	}
	if override.Generate.Output.Config != "" {
		result.Generate.Output.Config = override.Generate.Output.Config
	}

	if override.Server.Port != 0 {
		result.Server.Port = override.Server.Port
	}
	if override.Server.Environment != "" {
		result.Server.Environment = override.Server.Environment
	}

	if override.Database.URL != "" {
		result.Database.URL = override.Database.URL
	}

	if override.Pagination.DefaultLimit != 0 {
		result.Pagination.DefaultLimit = override.Pagination.DefaultLimit
	}
	if override.Pagination.MaxLimit != 0 {
		result.Pagination.MaxLimit = override.Pagination.MaxLimit
	}

	if len(override.Plugins.Global) > 0 {
		result.Plugins.Global = override.Plugins.Global
	}
	if len(override.Plugins.Route) > 0 {
		result.Plugins.Route = override.Plugins.Route
	}

	return &result
}

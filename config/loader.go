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

	for i := range config.Plugins {
		interpolatePluginConfig(config.Plugins[i].Config)
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

	if override.Codegen.Output.Models != "" {
		result.Codegen.Output.Models = override.Codegen.Output.Models
	}
	if override.Codegen.Output.Resources != "" {
		result.Codegen.Output.Resources = override.Codegen.Output.Resources
	}
	if override.Codegen.Output.DTOs != "" {
		result.Codegen.Output.DTOs = override.Codegen.Output.DTOs
	}
	if override.Codegen.Output.OpenAPI != "" {
		result.Codegen.Output.OpenAPI = override.Codegen.Output.OpenAPI
	}
	if override.Codegen.Output.Config != "" {
		result.Codegen.Output.Config = override.Codegen.Output.Config
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

	if len(override.Plugins) > 0 {
		result.Plugins = override.Plugins
	}

	return &result
}

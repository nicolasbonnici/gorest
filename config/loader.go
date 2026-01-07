package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// envVarRegex matches ${VAR} or ${VAR:-default} patterns
var envVarRegex = regexp.MustCompile(`\$\{([^}:]+)(:-([^}]*))?\}`)

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
	// Interpolate server config
	config.Server.Scheme = interpolateString(config.Server.Scheme)
	config.Server.Host = interpolateString(config.Server.Host)
	config.Server.Environment = interpolateString(config.Server.Environment)
	config.Server.CORSOrigins = interpolateString(config.Server.CORSOrigins)

	// Interpolate database config
	config.Database.URL = interpolateString(config.Database.URL)

	// Interpolate plugin configs
	for i := range config.Plugins {
		interpolatePluginConfig(config.Plugins[i].Config)
	}

	// Check for required database URL
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
		// Extract variable name and optional default value
		// Match groups: [0]=full match, [1]=variable name, [2]=:-default (with :-), [3]=default value
		matches := envVarRegex.FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}

		varName := matches[1]
		value := os.Getenv(varName)

		// If environment variable is set (even if empty string), use it
		if _, exists := os.LookupEnv(varName); exists {
			return value
		}

		// If environment variable is not set, check for default value
		// matches[2] contains ":-default" or empty string
		// matches[3] contains the default value (everything after :-)
		if len(matches) > 2 && matches[2] != "" {
			// Default value is present (matches[3] contains the default, which may be empty)
			return matches[3]
		}

		// No value and no default - return original match
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

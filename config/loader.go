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

// Load loads the configuration from .gorest.yaml file
// It supports environment-specific overrides via .gorest.{ENVIRONMENT}.yaml
// and environment variable interpolation via ${VAR} syntax
func Load(configPath string) (*Config, error) {
	// Find base config file
	baseConfigFile := filepath.Join(configPath, ".gorest.yaml")
	if _, err := os.Stat(baseConfigFile); os.IsNotExist(err) {
		return nil, fmt.Errorf(".gorest.yaml not found in %s - configuration file is required", configPath)
	}

	// Load base configuration
	baseConfig, err := loadConfigFile(baseConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load base config: %w", err)
	}

	// Check for environment-specific override
	environment := baseConfig.Server.Environment
	if environment == "" {
		environment = os.Getenv("ENVIRONMENT")
	}
	if environment == "" {
		environment = "development"
	}

	// Try to load environment-specific config
	envConfigFile := filepath.Join(configPath, fmt.Sprintf(".gorest.%s.yaml", environment))
	if _, err := os.Stat(envConfigFile); err == nil {
		envConfig, err := loadConfigFile(envConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load environment config: %w", err)
		}
		// Merge environment-specific config into base config
		baseConfig = mergeConfigs(baseConfig, envConfig)
	}

	// Set defaults for optional fields
	baseConfig.SetDefaults()

	// Interpolate environment variables
	if err := interpolateEnvVars(baseConfig); err != nil {
		return nil, fmt.Errorf("failed to interpolate environment variables: %w", err)
	}

	// Validate configuration
	if err := baseConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Warn about production settings
	warnings := baseConfig.WarnProductionSettings()
	for _, warning := range warnings {
		fmt.Fprintf(os.Stderr, "WARNING: %s\n", warning)
	}

	return baseConfig, nil
}

// loadConfigFile loads a YAML config file
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

// interpolateEnvVars replaces ${VAR} with environment variable values
func interpolateEnvVars(config *Config) error {
	// Interpolate database URL
	config.Database.URL = interpolateString(config.Database.URL)

	// Interpolate JWT secret
	config.Auth.JWT.Secret = interpolateString(config.Auth.JWT.Secret)

	// Validate that required environment variables were found
	if strings.HasPrefix(config.Database.URL, "${") && strings.HasSuffix(config.Database.URL, "}") {
		varName := strings.TrimSuffix(strings.TrimPrefix(config.Database.URL, "${"), "}")
		return fmt.Errorf("environment variable %s not found (required for database.url)", varName)
	}

	if strings.HasPrefix(config.Auth.JWT.Secret, "${") && strings.HasSuffix(config.Auth.JWT.Secret, "}") {
		varName := strings.TrimSuffix(strings.TrimPrefix(config.Auth.JWT.Secret, "${"), "}")
		return fmt.Errorf("environment variable %s not found (required for auth.jwt.secret)", varName)
	}

	return nil
}

// interpolateString replaces ${VAR} patterns with environment variable values
func interpolateString(s string) string {
	return envVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from ${VAR}
		varName := match[2 : len(match)-1]

		// Get environment variable value
		value := os.Getenv(varName)
		if value != "" {
			return value
		}

		// Return original if not found (will be validated later)
		return match
	})
}

// mergeConfigs merges override config into base config
// Override values take precedence over base values
func mergeConfigs(base, override *Config) *Config {
	result := *base

	// Merge Generate config
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

	// Merge Server config
	if override.Server.Port != 0 {
		result.Server.Port = override.Server.Port
	}
	if override.Server.Environment != "" {
		result.Server.Environment = override.Server.Environment
	}

	// Merge Database config
	if override.Database.URL != "" {
		result.Database.URL = override.Database.URL
	}

	// Merge Auth config
	if override.Auth.JWT.Secret != "" {
		result.Auth.JWT.Secret = override.Auth.JWT.Secret
	}
	if override.Auth.JWT.TTL != 0 {
		result.Auth.JWT.TTL = override.Auth.JWT.TTL
	}

	// Merge Pagination config
	if override.Pagination.DefaultLimit != 0 {
		result.Pagination.DefaultLimit = override.Pagination.DefaultLimit
	}
	if override.Pagination.MaxLimit != 0 {
		result.Pagination.MaxLimit = override.Pagination.MaxLimit
	}

	// Merge CORS config
	if override.CORS.Origins != nil {
		result.CORS.Origins = override.CORS.Origins
	}

	// Merge RateLimit config
	if override.RateLimit.RequestsPerSecond != 0 {
		result.RateLimit.RequestsPerSecond = override.RateLimit.RequestsPerSecond
	}
	if override.RateLimit.Burst != 0 {
		result.RateLimit.Burst = override.RateLimit.Burst
	}
	// Note: Enabled is boolean, so we always take override value
	result.RateLimit.Enabled = override.RateLimit.Enabled

	return &result
}

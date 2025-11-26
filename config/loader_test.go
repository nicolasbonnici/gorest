package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_ValidBaseConfig(t *testing.T) {
	tmpDir := t.TempDir()

	configYAML := `
server:
  port: 3000
  environment: development
database:
  url: postgres://localhost/testdb
pagination:
  default_limit: 10
  max_limit: 100
generate:
  output:
    models: "generated/models"
    resources: "generated/resources"
    dtos: "generated/dtos"
plugins: []
`

	os.WriteFile(filepath.Join(tmpDir, "gorest.yaml"), []byte(configYAML), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("Expected port 3000, got %d", cfg.Server.Port)
	}

	if cfg.Database.URL != "postgres://localhost/testdb" {
		t.Errorf("Expected database URL to match, got %s", cfg.Database.URL)
	}

	if cfg.Server.Environment != "development" {
		t.Errorf("Expected environment 'development', got %s", cfg.Server.Environment)
	}
}

func TestLoad_MissingBaseConfig(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := Load(tmpDir)
	if err == nil {
		t.Fatal("Expected error for missing gorest.yaml")
	}

	if !strings.Contains(err.Error(), "gorest.yaml not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	invalidYAML := `
server:
  port: invalid_port
  this is not valid yaml
`

	os.WriteFile(filepath.Join(tmpDir, "gorest.yaml"), []byte(invalidYAML), 0644)

	_, err := Load(tmpDir)
	if err == nil {
		t.Fatal("Expected error for invalid YAML")
	}

	if !strings.Contains(err.Error(), "failed to parse YAML") {
		t.Errorf("Expected YAML parse error, got: %v", err)
	}
}

func TestLoad_EnvironmentOverride(t *testing.T) {
	tmpDir := t.TempDir()

	baseConfig := `
server:
  port: 3000
  environment: production
database:
  url: postgres://localhost/db
pagination:
  default_limit: 10
  max_limit: 100
generate:
  output:
    models: "models"
    resources: "resources"
    dtos: "dtos"
plugins: []
`

	prodConfig := `
server:
  port: 8080
database:
  url: postgres://prod-db/db
pagination:
  default_limit: 20
`

	os.WriteFile(filepath.Join(tmpDir, "gorest.yaml"), []byte(baseConfig), 0644)
	os.WriteFile(filepath.Join(tmpDir, "gorest.production.yaml"), []byte(prodConfig), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Expected port 8080 from environment override, got %d", cfg.Server.Port)
	}

	if cfg.Database.URL != "postgres://prod-db/db" {
		t.Errorf("Expected overridden database URL, got %s", cfg.Database.URL)
	}

	if cfg.Pagination.DefaultLimit != 20 {
		t.Errorf("Expected pagination limit 20, got %d", cfg.Pagination.DefaultLimit)
	}
}

func TestLoad_EnvironmentFromEnvVar(t *testing.T) {
	tmpDir := t.TempDir()

	baseConfig := `
server:
  port: 3000
database:
  url: postgres://localhost/db
pagination:
  default_limit: 10
  max_limit: 100
generate:
  output:
    models: "models"
    resources: "resources"
    dtos: "dtos"
plugins: []
`

	stagingConfig := `
server:
  port: 4000
`

	os.WriteFile(filepath.Join(tmpDir, "gorest.yaml"), []byte(baseConfig), 0644)
	os.WriteFile(filepath.Join(tmpDir, "gorest.staging.yaml"), []byte(stagingConfig), 0644)

	os.Setenv("ENVIRONMENT", "staging")
	defer os.Unsetenv("ENVIRONMENT")

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Server.Port != 4000 {
		t.Errorf("Expected port 4000 from staging config, got %d", cfg.Server.Port)
	}
}

func TestInterpolateEnvVars_Success(t *testing.T) {
	os.Setenv("DB_HOST", "prodserver")
	os.Setenv("DB_PASS", "secret123")
	defer os.Unsetenv("DB_HOST")
	defer os.Unsetenv("DB_PASS")

	cfg := &Config{
		Database: DatabaseConfig{
			URL: "postgres://user:${DB_PASS}@${DB_HOST}/mydb",
		},
	}

	err := interpolateEnvVars(cfg)
	if err != nil {
		t.Fatalf("interpolateEnvVars() failed: %v", err)
	}

	expected := "postgres://user:secret123@prodserver/mydb"
	if cfg.Database.URL != expected {
		t.Errorf("Expected %q, got %q", expected, cfg.Database.URL)
	}
}

func TestInterpolateEnvVars_MissingRequired(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			URL: "${MISSING_DB_URL}",
		},
	}

	err := interpolateEnvVars(cfg)
	if err == nil {
		t.Fatal("Expected error for missing required environment variable")
	}

	if !strings.Contains(err.Error(), "MISSING_DB_URL") {
		t.Errorf("Error should mention missing variable name, got: %v", err)
	}
}

func TestInterpolateEnvVars_PluginConfig(t *testing.T) {
	os.Setenv("JWT_SECRET", "super-secret-key-12345678901234567890")
	os.Setenv("API_KEY", "test-api-key")
	defer os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("API_KEY")

	cfg := &Config{
		Database: DatabaseConfig{URL: "postgres://localhost/db"},
		Plugins: PluginsConfig{
			{
				Name:    "auth",
				Enabled: true,
				Config: map[string]interface{}{
					"jwt_secret": "${JWT_SECRET}",
					"api_key":    "${API_KEY}",
				},
			},
		},
	}

	err := interpolateEnvVars(cfg)
	if err != nil {
		t.Fatalf("interpolateEnvVars() failed: %v", err)
	}

	secret := cfg.Plugins[0].Config["jwt_secret"].(string)
	if secret != "super-secret-key-12345678901234567890" {
		t.Errorf("JWT secret not interpolated correctly, got: %s", secret)
	}

	apiKey := cfg.Plugins[0].Config["api_key"].(string)
	if apiKey != "test-api-key" {
		t.Errorf("API key not interpolated correctly, got: %s", apiKey)
	}
}

func TestInterpolateEnvVars_GlobalPluginConfig(t *testing.T) {
	os.Setenv("CORS_ORIGIN", "https://example.com")
	defer os.Unsetenv("CORS_ORIGIN")

	cfg := &Config{
		Database: DatabaseConfig{URL: "postgres://localhost/db"},
		Plugins: PluginsConfig{
			{
				Name:    "cors",
				Enabled: true,
				Config: map[string]interface{}{
					"origins": "${CORS_ORIGIN}",
				},
			},
		},
	}

	err := interpolateEnvVars(cfg)
	if err != nil {
		t.Fatalf("interpolateEnvVars() failed: %v", err)
	}

	origin := cfg.Plugins[0].Config["origins"].(string)
	if origin != "https://example.com" {
		t.Errorf("CORS origin not interpolated correctly, got: %s", origin)
	}
}

func TestInterpolateEnvVars_PartialInterpolation(t *testing.T) {
	os.Setenv("DB_NAME", "production")
	defer os.Unsetenv("DB_NAME")

	cfg := &Config{
		Database: DatabaseConfig{
			URL: "postgres://localhost/${DB_NAME}",
		},
	}

	err := interpolateEnvVars(cfg)
	if err != nil {
		t.Fatalf("interpolateEnvVars() failed: %v", err)
	}

	expected := "postgres://localhost/production"
	if cfg.Database.URL != expected {
		t.Errorf("Expected %q, got %q", expected, cfg.Database.URL)
	}
}

func TestInterpolateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "single variable",
			input:    "${VAR1}",
			envVars:  map[string]string{"VAR1": "value1"},
			expected: "value1",
		},
		{
			name:     "multiple variables",
			input:    "${VAR1}:${VAR2}",
			envVars:  map[string]string{"VAR1": "host", "VAR2": "port"},
			expected: "host:port",
		},
		{
			name:     "no variables",
			input:    "static-value",
			envVars:  map[string]string{},
			expected: "static-value",
		},
		{
			name:     "missing variable",
			input:    "${MISSING}",
			envVars:  map[string]string{},
			expected: "${MISSING}",
		},
		{
			name:     "mixed content",
			input:    "prefix-${VAR}-suffix",
			envVars:  map[string]string{"VAR": "middle"},
			expected: "prefix-middle-suffix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result := interpolateString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestMergeConfigs(t *testing.T) {
	base := &Config{
		Server: ServerConfig{
			Port:        3000,
			Environment: "development",
		},
		Database: DatabaseConfig{
			URL: "postgres://localhost/dev",
		},
		Pagination: PaginationConfig{
			DefaultLimit: 10,
			MaxLimit:     100,
		},
		Plugins: PluginsConfig{
			{Name: "cors", Enabled: true},
		},
		Generate: GenerateConfig{
			Output: OutputConfig{
				Models:    "models",
				Resources: "resources",
			},
		},
	}

	override := &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Database: DatabaseConfig{
			URL: "postgres://prod/db",
		},
		Pagination: PaginationConfig{
			DefaultLimit: 20,
		},
		Plugins: PluginsConfig{
			{Name: "auth", Enabled: true},
		},
	}

	result := mergeConfigs(base, override)

	if result.Server.Port != 8080 {
		t.Errorf("Port should be overridden, got %d", result.Server.Port)
	}

	if result.Server.Environment != "development" {
		t.Errorf("Environment should remain from base, got %s", result.Server.Environment)
	}

	if result.Database.URL != "postgres://prod/db" {
		t.Errorf("Database URL should be overridden, got %s", result.Database.URL)
	}

	if result.Pagination.DefaultLimit != 20 {
		t.Errorf("DefaultLimit should be overridden, got %d", result.Pagination.DefaultLimit)
	}

	if result.Pagination.MaxLimit != 100 {
		t.Errorf("MaxLimit should remain from base, got %d", result.Pagination.MaxLimit)
	}

	if len(result.Plugins) != 1 {
		t.Errorf("Plugins should be replaced, got %d", len(result.Plugins))
	}

	if result.Plugins[0].Name != "auth" {
		t.Errorf("Plugin should be from override, got %s", result.Plugins[0].Name)
	}
}

func TestMergeConfigs_EmptyOverride(t *testing.T) {
	base := &Config{
		Server: ServerConfig{
			Port:        3000,
			Environment: "development",
		},
		Database: DatabaseConfig{
			URL: "postgres://localhost/dev",
		},
	}

	override := &Config{}

	result := mergeConfigs(base, override)

	if result.Server.Port != 3000 {
		t.Errorf("Port should remain from base, got %d", result.Server.Port)
	}

	if result.Database.URL != "postgres://localhost/dev" {
		t.Errorf("Database URL should remain from base, got %s", result.Database.URL)
	}
}

func TestMergeConfigs_PluginOverride(t *testing.T) {
	base := &Config{
		Plugins: PluginsConfig{
			{Name: "cors", Enabled: true},
			{Name: "logger", Enabled: true},
			{Name: "auth", Enabled: false},
		},
	}

	override := &Config{
		Plugins: PluginsConfig{
			{Name: "ratelimit", Enabled: true},
		},
	}

	result := mergeConfigs(base, override)

	if len(result.Plugins) != 1 {
		t.Errorf("Plugins should be replaced completely, got %d", len(result.Plugins))
	}

	if result.Plugins[0].Name != "ratelimit" {
		t.Errorf("Plugin should be from override, got %s", result.Plugins[0].Name)
	}
}

func TestLoadConfigFile_Success(t *testing.T) {
	tmpDir := t.TempDir()

	validYAML := `
server:
  port: 5000
database:
  url: postgres://localhost/test
`

	configPath := filepath.Join(tmpDir, "test.yaml")
	os.WriteFile(configPath, []byte(validYAML), 0644)

	cfg, err := loadConfigFile(configPath)
	if err != nil {
		t.Fatalf("loadConfigFile() failed: %v", err)
	}

	if cfg.Server.Port != 5000 {
		t.Errorf("Expected port 5000, got %d", cfg.Server.Port)
	}
}

func TestLoadConfigFile_FileNotFound(t *testing.T) {
	_, err := loadConfigFile("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}
}

func TestLoad_AppliesDefaults(t *testing.T) {
	tmpDir := t.TempDir()

	minimalConfig := `
database:
  url: postgres://localhost/db
generate:
  output:
    models: "models"
    resources: "resources"
    dtos: "dtos"
plugins: []
`

	os.WriteFile(filepath.Join(tmpDir, "gorest.yaml"), []byte(minimalConfig), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("Expected default port 3000, got %d", cfg.Server.Port)
	}

	if cfg.Server.Environment != "development" {
		t.Errorf("Expected default environment 'development', got %s", cfg.Server.Environment)
	}

	if cfg.Pagination.DefaultLimit != 10 {
		t.Errorf("Expected default pagination limit 10, got %d", cfg.Pagination.DefaultLimit)
	}

	if cfg.Pagination.MaxLimit != 1000 {
		t.Errorf("Expected default max limit 1000, got %d", cfg.Pagination.MaxLimit)
	}
}

func TestLoad_ValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()

	invalidConfig := `
server:
  port: -1
database:
  url: postgres://localhost/db
pagination:
  default_limit: 10
  max_limit: 100
generate:
  output:
    models: "models"
    resources: "resources"
    dtos: "dtos"
plugins: []
`

	os.WriteFile(filepath.Join(tmpDir, "gorest.yaml"), []byte(invalidConfig), 0644)

	_, err := Load(tmpDir)
	if err == nil {
		t.Fatal("Expected validation error for invalid port")
	}

	if !strings.Contains(err.Error(), "invalid configuration") {
		t.Errorf("Expected validation error, got: %v", err)
	}
}

package config

import (
	"strings"
	"testing"
)

func TestValidate_MissingDatabaseURL(t *testing.T) {
	cfg := &Config{
		Server:     ServerConfig{Port: 8000},
		Pagination: PaginationConfig{DefaultLimit: 10, MaxLimit: 100},
		Codegen: CodegenConfig{
			Output: OutputConfig{
				Models:    "models",
				Resources: "resources",
				DTOs:      "dtos",
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation error for missing database URL")
	}

	if !strings.Contains(err.Error(), "database.url") {
		t.Errorf("Error should mention database.url, got: %v", err)
	}
}

func TestValidate_InvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"zero port", 0},
		{"negative port", -1},
		{"port too high", 70000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:     ServerConfig{Port: tt.port},
				Database:   DatabaseConfig{URL: "postgres://localhost/db"},
				Pagination: PaginationConfig{DefaultLimit: 10, MaxLimit: 100},
				Codegen: CodegenConfig{
					Output: OutputConfig{
						Models:    "models",
						Resources: "resources",
						DTOs:      "dtos",
					},
				},
			}

			err := cfg.Validate()
			if err == nil {
				t.Errorf("Expected validation error for port %d", tt.port)
			}

			if !strings.Contains(err.Error(), "server.port") {
				t.Errorf("Error should mention server.port, got: %v", err)
			}
		})
	}
}

func TestValidate_ValidPort(t *testing.T) {
	tests := []int{1, 80, 8000, 8080, 65535}

	for _, port := range tests {
		t.Run("port "+string(rune(port)), func(t *testing.T) {
			cfg := &Config{
				Server:     ServerConfig{Port: port},
				Database:   DatabaseConfig{URL: "postgres://localhost/db"},
				Pagination: PaginationConfig{DefaultLimit: 10, MaxLimit: 100},
				Codegen: CodegenConfig{
					Output: OutputConfig{
						Models:    "models",
						Resources: "resources",
						DTOs:      "dtos",
					},
				},
			}

			err := cfg.Validate()
			if err != nil && strings.Contains(err.Error(), "server.port") {
				t.Errorf("Port %d should be valid, got error: %v", port, err)
			}
		})
	}
}

func TestValidate_AuthPluginJWTSecret(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "valid secret",
			secret:    "this-is-a-valid-secret-with-32-characters-minimum",
			shouldErr: false,
		},
		{
			name:      "too short",
			secret:    "short",
			shouldErr: true,
			errMsg:    "must be at least 32 characters",
		},
		{
			name:      "empty",
			secret:    "",
			shouldErr: true,
			errMsg:    "is required",
		},
		{
			name:      "exactly 32 characters",
			secret:    "12345678901234567890123456789012",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:     ServerConfig{Port: 8000},
				Database:   DatabaseConfig{URL: "postgres://localhost/db"},
				Pagination: PaginationConfig{DefaultLimit: 10, MaxLimit: 100},
				Codegen: CodegenConfig{
					Output: OutputConfig{
						Models:    "models",
						Resources: "resources",
						DTOs:      "dtos",
					},
				},
				Plugins: PluginsConfig{
					{
						Name:    "auth",
						Enabled: true,
						Config: map[string]interface{}{
							"jwt_secret": tt.secret,
						},
					},
				},
			}

			err := cfg.Validate()
			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected validation error")
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidate_AuthPluginDisabled(t *testing.T) {
	cfg := &Config{
		Server:     ServerConfig{Port: 8000},
		Database:   DatabaseConfig{URL: "postgres://localhost/db"},
		Pagination: PaginationConfig{DefaultLimit: 10, MaxLimit: 100},
		Codegen: CodegenConfig{
			Output: OutputConfig{
				Models:    "models",
				Resources: "resources",
				DTOs:      "dtos",
			},
		},
		Plugins: PluginsConfig{
			{
				Name:    "auth",
				Enabled: false,
				Config: map[string]interface{}{
					"jwt_secret": "short",
				},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Disabled auth plugin should not be validated, got: %v", err)
	}
}

func TestValidate_AuthPluginMissingSecret(t *testing.T) {
	cfg := &Config{
		Server:     ServerConfig{Port: 8000},
		Database:   DatabaseConfig{URL: "postgres://localhost/db"},
		Pagination: PaginationConfig{DefaultLimit: 10, MaxLimit: 100},
		Codegen: CodegenConfig{
			Output: OutputConfig{
				Models:    "models",
				Resources: "resources",
				DTOs:      "dtos",
			},
		},
		Plugins: PluginsConfig{
			{
				Name:    "auth",
				Enabled: true,
				Config:  map[string]interface{}{},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation error for missing jwt_secret")
	}

	if !strings.Contains(err.Error(), "jwt_secret is required") {
		t.Errorf("Error should mention jwt_secret is required, got: %v", err)
	}
}

func TestValidate_PaginationLimits(t *testing.T) {
	tests := []struct {
		name         string
		defaultLimit int
		maxLimit     int
		shouldErr    bool
		errMsg       string
	}{
		{"valid", 10, 100, false, ""},
		{"equal limits", 50, 50, false, ""},
		{"default > max", 100, 50, true, "cannot exceed max_limit"},
		{"zero default", 0, 100, true, "must be positive"},
		{"negative default", -10, 100, true, "must be positive"},
		{"zero max", 10, 0, true, "must be positive"},
		{"negative max", 10, -5, true, "must be positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:   ServerConfig{Port: 8000},
				Database: DatabaseConfig{URL: "postgres://localhost/db"},
				Pagination: PaginationConfig{
					DefaultLimit: tt.defaultLimit,
					MaxLimit:     tt.maxLimit,
				},
				Codegen: CodegenConfig{
					Output: OutputConfig{
						Models:    "models",
						Resources: "resources",
						DTOs:      "dtos",
					},
				},
			}

			err := cfg.Validate()
			if tt.shouldErr {
				if err == nil {
					t.Error("Expected validation error")
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %v", tt.errMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidate_MissingGenerateOutputPaths(t *testing.T) {
	tests := []struct {
		name      string
		models    string
		resources string
		dtos      string
		errMsg    string
	}{
		{"missing models", "", "resources", "dtos", "codegen.output.models"},
		{"missing resources", "models", "", "dtos", "codegen.output.resources"},
		{"missing dtos", "models", "resources", "", "codegen.output.dtos"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Server:     ServerConfig{Port: 8000},
				Database:   DatabaseConfig{URL: "postgres://localhost/db"},
				Pagination: PaginationConfig{DefaultLimit: 10, MaxLimit: 100},
				Codegen: CodegenConfig{
					Output: OutputConfig{
						Models:    tt.models,
						Resources: tt.resources,
						DTOs:      tt.dtos,
					},
				},
			}

			err := cfg.Validate()
			if err == nil {
				t.Fatal("Expected validation error")
			}

			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing %q, got %v", tt.errMsg, err)
			}
		})
	}
}

func TestValidate_Success(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port:        3000,
			Environment: "production",
		},
		Database: DatabaseConfig{
			URL: "postgres://localhost/db",
		},
		Pagination: PaginationConfig{
			DefaultLimit: 10,
			MaxLimit:     100,
		},
		Codegen: CodegenConfig{
			Output: OutputConfig{
				Models:    "models",
				Resources: "resources",
				DTOs:      "dtos",
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Valid config should not error, got: %v", err)
	}
}

func TestSetDefaults(t *testing.T) {
	cfg := &Config{}

	cfg.SetDefaults()

	if cfg.Server.Port != 8000 {
		t.Errorf("Expected default port 8000, got %d", cfg.Server.Port)
	}

	if cfg.Server.Environment != "development" {
		t.Errorf("Expected default environment 'development', got %q", cfg.Server.Environment)
	}

	if cfg.Pagination.DefaultLimit != 10 {
		t.Errorf("Expected default pagination limit 10, got %d", cfg.Pagination.DefaultLimit)
	}

	if cfg.Pagination.MaxLimit != 1000 {
		t.Errorf("Expected max pagination limit 1000, got %d", cfg.Pagination.MaxLimit)
	}

	if cfg.Codegen.Output.Models != "generated/models" {
		t.Errorf("Expected default models path, got %s", cfg.Codegen.Output.Models)
	}

	if cfg.Codegen.Output.Resources != "generated/resources" {
		t.Errorf("Expected default resources path, got %s", cfg.Codegen.Output.Resources)
	}

	if cfg.Codegen.Output.DTOs != "generated/dtos" {
		t.Errorf("Expected default dtos path, got %s", cfg.Codegen.Output.DTOs)
	}

	if cfg.Codegen.Output.OpenAPI != "generated/openapi" {
		t.Errorf("Expected default openapi path, got %s", cfg.Codegen.Output.OpenAPI)
	}

	if cfg.Codegen.Output.Config != "generated/config" {
		t.Errorf("Expected default config path, got %s", cfg.Codegen.Output.Config)
	}
}

func TestSetDefaults_PreservesExistingValues(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port:        8080,
			Environment: "production",
		},
		Pagination: PaginationConfig{
			DefaultLimit: 20,
			MaxLimit:     500,
		},
		Codegen: CodegenConfig{
			Output: OutputConfig{
				Models:    "custom/models",
				Resources: "custom/resources",
			},
		},
	}

	cfg.SetDefaults()

	if cfg.Server.Port != 8080 {
		t.Errorf("Port should not be overridden, got %d", cfg.Server.Port)
	}

	if cfg.Server.Environment != "production" {
		t.Errorf("Environment should not be overridden, got %s", cfg.Server.Environment)
	}

	if cfg.Pagination.DefaultLimit != 20 {
		t.Errorf("DefaultLimit should not be overridden, got %d", cfg.Pagination.DefaultLimit)
	}

	if cfg.Pagination.MaxLimit != 500 {
		t.Errorf("MaxLimit should not be overridden, got %d", cfg.Pagination.MaxLimit)
	}

	if cfg.Codegen.Output.Models != "custom/models" {
		t.Errorf("Models path should not be overridden, got %s", cfg.Codegen.Output.Models)
	}

	if cfg.Codegen.Output.Resources != "custom/resources" {
		t.Errorf("Resources path should not be overridden, got %s", cfg.Codegen.Output.Resources)
	}

	if cfg.Codegen.Output.DTOs != "generated/dtos" {
		t.Errorf("DTOs path should be set to default when empty, got %s", cfg.Codegen.Output.DTOs)
	}
}

func TestSetDefaults_PartialConfig(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port: 4000,
		},
	}

	cfg.SetDefaults()

	if cfg.Server.Port != 4000 {
		t.Errorf("Existing port should be preserved, got %d", cfg.Server.Port)
	}

	if cfg.Server.Environment != "development" {
		t.Errorf("Environment should be set to default, got %s", cfg.Server.Environment)
	}

	if cfg.Pagination.DefaultLimit != 10 {
		t.Errorf("Pagination defaults should be set, got %d", cfg.Pagination.DefaultLimit)
	}
}

func TestValidate_MultipleAuthPlugins(t *testing.T) {
	cfg := &Config{
		Server:     ServerConfig{Port: 8000},
		Database:   DatabaseConfig{URL: "postgres://localhost/db"},
		Pagination: PaginationConfig{DefaultLimit: 10, MaxLimit: 100},
		Codegen: CodegenConfig{
			Output: OutputConfig{
				Models:    "models",
				Resources: "resources",
				DTOs:      "dtos",
			},
		},
		Plugins: PluginsConfig{
			{
				Name:    "auth",
				Enabled: true,
				Config: map[string]interface{}{
					"jwt_secret": "valid-secret-key-with-32-chars-minimum",
				},
			},
			{
				Name:    "custom-auth",
				Enabled: true,
				Config:  map[string]interface{}{},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Only auth plugin should be validated, got: %v", err)
	}
}

func TestValidate_AuthPluginWrongType(t *testing.T) {
	cfg := &Config{
		Server:     ServerConfig{Port: 8000},
		Database:   DatabaseConfig{URL: "postgres://localhost/db"},
		Pagination: PaginationConfig{DefaultLimit: 10, MaxLimit: 100},
		Codegen: CodegenConfig{
			Output: OutputConfig{
				Models:    "models",
				Resources: "resources",
				DTOs:      "dtos",
			},
		},
		Plugins: PluginsConfig{
			{
				Name:    "auth",
				Enabled: true,
				Config: map[string]interface{}{
					"jwt_secret": 123,
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation error for wrong type")
	}

	if !strings.Contains(err.Error(), "jwt_secret is required") {
		t.Errorf("Error should mention jwt_secret is required, got: %v", err)
	}
}

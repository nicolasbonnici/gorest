package config

import (
	"fmt"
	"strings"
)

type Config struct {
	Codegen    CodegenConfig    `yaml:"codegen"`
	API        APIConfig        `yaml:"api"`
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Pagination PaginationConfig `yaml:"pagination"`
	Plugins    PluginsConfig    `yaml:"plugins"`
	RBAC       RBACConfig       `yaml:"rbac"`
	Auth       AuthConfig       `yaml:"auth"`
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

type APIConfig struct {
	Versioning VersioningConfig `yaml:"versioning"`
}

type VersioningConfig struct {
	Enabled bool   `yaml:"enabled"`
	Version string `yaml:"version"`
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

type RBACConfig struct {
	Enabled            bool                `yaml:"enabled"`
	DefaultPolicy      string              `yaml:"default_policy"`
	SuperuserRole      string              `yaml:"superuser_role"`
	RoleHierarchy      map[string][]string `yaml:"role_hierarchy"`
	DefaultFieldPolicy string              `yaml:"default_field_policy"`
	StrictValidation   bool                `yaml:"strict_validation"`
	CacheEnabled       bool                `yaml:"cache_enabled"`
	CacheTTL           int                 `yaml:"cache_ttl"`
}

type AuthConfig struct {
	Enabled   bool   `yaml:"enabled"`
	JWTSecret string `yaml:"jwt_secret"`
	JWTTTL    int    `yaml:"jwt_ttl"`
}

func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("database.url is required")
	}

	// Validate auth configuration
	if c.Auth.Enabled {
		if c.Auth.JWTSecret == "" {
			return fmt.Errorf("auth.jwt_secret is required when auth is enabled")
		}
		if len(c.Auth.JWTSecret) < 32 {
			return fmt.Errorf("auth.jwt_secret must be at least 32 characters long for security")
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

	// Validate RBAC configuration
	if c.RBAC.DefaultPolicy != "" && c.RBAC.DefaultPolicy != "deny_all" && c.RBAC.DefaultPolicy != "allow_all" {
		return fmt.Errorf("rbac.default_policy must be 'deny_all' or 'allow_all'")
	}

	if c.RBAC.DefaultFieldPolicy != "" && c.RBAC.DefaultFieldPolicy != "deny" && c.RBAC.DefaultFieldPolicy != "allow" {
		return fmt.Errorf("rbac.default_field_policy must be 'deny' or 'allow'")
	}

	if c.RBAC.CacheEnabled && c.RBAC.CacheTTL <= 0 {
		return fmt.Errorf("rbac.cache_ttl must be greater than 0 when cache is enabled")
	}

	// Validate role hierarchy for cycles
	if err := validateRoleHierarchy(c.RBAC.RoleHierarchy); err != nil {
		return fmt.Errorf("rbac.role_hierarchy: %w", err)
	}

	return nil
}

// validateRoleHierarchy checks for circular dependencies in role hierarchy
func validateRoleHierarchy(hierarchy map[string][]string) error {
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	var hasCycle func(role string) bool
	hasCycle = func(role string) bool {
		visited[role] = true
		recursionStack[role] = true

		for _, child := range hierarchy[role] {
			if !visited[child] {
				if hasCycle(child) {
					return true
				}
			} else if recursionStack[child] {
				return true
			}
		}

		recursionStack[role] = false
		return false
	}

	for role := range hierarchy {
		if !visited[role] {
			if hasCycle(role) {
				return fmt.Errorf("circular dependency detected involving role '%s'", role)
			}
		}
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

	// API defaults
	if c.API.Versioning.Version == "" {
		c.API.Versioning.Version = "v1"
	}
	// Versioning.Enabled defaults to true unless explicitly set to false

	// RBAC defaults
	if c.RBAC.DefaultPolicy == "" {
		c.RBAC.DefaultPolicy = "deny_all"
	}
	if c.RBAC.SuperuserRole == "" {
		c.RBAC.SuperuserRole = "admin"
	}
	if c.RBAC.RoleHierarchy == nil {
		c.RBAC.RoleHierarchy = make(map[string][]string)
	}
	if c.RBAC.DefaultFieldPolicy == "" {
		c.RBAC.DefaultFieldPolicy = "deny"
	}
	if c.RBAC.CacheTTL == 0 {
		c.RBAC.CacheTTL = 300 // 5 minutes
	}
	// CacheEnabled defaults to false unless explicitly set
	// StrictValidation defaults to false unless explicitly set

	// Auth defaults
	if c.Auth.JWTTTL == 0 {
		c.Auth.JWTTTL = 900 // Default 15 minutes
	}
	// Auth.Enabled defaults to false unless explicitly set
}

package config

import (
	"fmt"
	"strings"
)

// DefaultBodyLimit caps the request body when server.body_limit is unset. A
// plugin accepting larger uploads needs this raised too: Fiber enforces it
// before routing, so the plugin never sees the request.
const DefaultBodyLimit = 4 << 20

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
	// BodyLimit is in bytes; 0 uses DefaultBodyLimit.
	BodyLimit int `yaml:"body_limit"`
}

type DatabaseConfig struct {
	URL  string       `yaml:"url"`
	Pool DBPoolConfig `yaml:"pool"`
}

// DBPoolConfig exposes connection-pool tuning. Zero values fall back to the
// framework defaults applied in gorest.go (see database.PoolConfig for the
// low-level knobs the driver consumes).
type DBPoolConfig struct {
	// MaxOpen caps concurrent open connections. 0 uses the GoREST default.
	MaxOpen int `yaml:"max_open"`
	// MaxIdle caps idle connections kept warm. 0 uses the driver default.
	MaxIdle int `yaml:"max_idle"`
}

type PaginationConfig struct {
	DefaultLimit int `yaml:"default_limit"`
	MaxLimit     int `yaml:"max_limit"`
	// Count is "exact" (default), "estimate" or "none". See crud.CountMode.
	// Clients can always opt out of a total per request with ?count=false.
	Count string `yaml:"count"`
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
	Enabled    bool   `yaml:"enabled"`
	JWTSecret  string `yaml:"jwt_secret"`
	JWTTTL     int    `yaml:"jwt_ttl"`
	RefreshTTL int    `yaml:"refresh_ttl"`

	// Throttle for the credential endpoints. The global server limiter is sized
	// for traffic floods and is far too loose to stop password guessing: a few
	// attempts per minute is invisible to it but is exactly what a brute force
	// looks like. Set LoginRateLimit to a negative number to switch this off.
	LoginRateLimit  int `yaml:"login_rate_limit"`
	LoginRateWindow int `yaml:"login_rate_window"`

	// Password policy applied at registration. Length is the control that
	// matters (NIST SP 800-63B); PasswordBlocklist adds screening against the
	// passwords that are guessed first, which length alone lets through.
	// Defaults to password.DefaultMinLength when zero, and the blocklist is on
	// unless explicitly disabled.
	PasswordMinLength int   `yaml:"password_min_length"`
	PasswordBlocklist *bool `yaml:"password_blocklist"`
}

// PasswordBlocklistEnabled reports whether common-password screening is on.
// The field is a pointer so an absent key means "on" while `false` means off;
// a plain bool could not tell the two apart.
func (a AuthConfig) PasswordBlocklistEnabled() bool {
	return a.PasswordBlocklist == nil || *a.PasswordBlocklist
}

// validateProductionHardening refuses to boot a production server that is
// configured like a development one. Each of these is a setting that is
// harmless on localhost and a real exposure on a public host, and every one of
// them is easy to carry into production by forgetting to change it.
func (c *Config) validateProductionHardening() error {
	if !strings.EqualFold(c.Server.Environment, "production") {
		return nil
	}

	for _, origin := range strings.Split(c.Server.CORSOrigins, ",") {
		if strings.TrimSpace(origin) == "*" {
			return fmt.Errorf("server.cors_origins cannot be \"*\" in production: name the origins that may read authenticated responses")
		}
	}

	if !c.Server.RateLimitEnabled {
		return fmt.Errorf("server.ratelimit_enabled must be true in production: without it the API has no brute-force or flood protection")
	}

	if !strings.EqualFold(c.Server.Scheme, "https") {
		return fmt.Errorf("server.scheme must be https in production, got %q: bearer tokens over plain http are readable in transit", c.Server.Scheme)
	}

	if c.Auth.Enabled {
		// Placeholders long enough to clear the 32-character floor still ship
		// as secrets in more deployments than anyone would like.
		lowered := strings.ToLower(c.Auth.JWTSecret)
		for _, marker := range []string{"change", "example", "placeholder", "your-secret", "test-secret", "dev-secret", "secret-key"} {
			if strings.Contains(lowered, marker) {
				return fmt.Errorf("auth.jwt_secret looks like a placeholder (contains %q); generate a random secret for production", marker)
			}
		}
	}

	return nil
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
		if c.Auth.RefreshTTL < 0 {
			return fmt.Errorf("auth.refresh_ttl cannot be negative")
		}
		// A refresh token that outlives no access token is useless: it would expire
		// before it ever gets a chance to mint a replacement.
		if c.Auth.RefreshTTL > 0 && c.Auth.JWTTTL > 0 && c.Auth.RefreshTTL <= c.Auth.JWTTTL {
			return fmt.Errorf("auth.refresh_ttl must be greater than auth.jwt_ttl")
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
	switch c.Pagination.Count {
	case "", "exact", "estimate", "none":
	default:
		return fmt.Errorf("pagination.count must be one of exact, estimate, none")
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}

	if err := c.validateProductionHardening(); err != nil {
		return err
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
	if c.Server.BodyLimit == 0 {
		c.Server.BodyLimit = DefaultBodyLimit
	}

	if c.Pagination.DefaultLimit == 0 {
		c.Pagination.DefaultLimit = 10
	}
	if c.Pagination.MaxLimit == 0 {
		c.Pagination.MaxLimit = 1000
	}
	if c.Pagination.Count == "" {
		c.Pagination.Count = "exact"
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
	if c.Auth.LoginRateLimit == 0 {
		c.Auth.LoginRateLimit = 10
	}
	if c.Auth.LoginRateWindow == 0 {
		c.Auth.LoginRateWindow = 300
	}
	// Auth.Enabled defaults to false unless explicitly set
}

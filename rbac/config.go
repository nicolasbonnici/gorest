package rbac

import (
	"fmt"
)

// Config holds RBAC configuration
type Config struct {
	// DefaultPolicy sets the default authorization policy
	// Can be "deny_all" (deny by default) or "allow_all" (allow by default)
	DefaultPolicy Policy `yaml:"default_policy" json:"default_policy"`

	// SuperuserRole is the role that bypasses all permission checks
	// Default: "admin"
	SuperuserRole string `yaml:"superuser_role" json:"superuser_role"`

	// RoleHierarchy defines parent-child relationships between roles
	// Key is the parent role, value is array of child roles
	// Example: "admin": ["editor", "user"] means admin inherits editor and user permissions
	RoleHierarchy map[string][]string `yaml:"role_hierarchy" json:"role_hierarchy"`

	// CacheEnabled enables permission caching for better performance
	CacheEnabled bool `yaml:"cache_enabled" json:"cache_enabled"`

	// CacheTTL is the time-to-live for cached permissions in seconds
	CacheTTL int `yaml:"cache_ttl" json:"cache_ttl"`

	// StrictMode causes initialization to fail on invalid annotations
	// If false, invalid annotations are logged and ignored
	StrictMode bool `yaml:"strict_mode" json:"strict_mode"`

	// DefaultFieldPolicy determines what happens when a field has no rbac tag
	// Can be "deny" (no access) or "allow" (full access)
	DefaultFieldPolicy string `yaml:"default_field_policy" json:"default_field_policy"`

	// StrictValidation controls whether zero values are validated during write operations.
	// When true, ALL fields (including zero values) are validated against permissions.
	// When false (default), zero values bypass validation for backwards compatibility.
	//
	// Security Note: Setting this to false may allow attackers to bypass validation by
	// setting restricted fields to their zero values (empty string, 0, false, nil, etc.).
	// It is recommended to enable StrictValidation in production environments.
	StrictValidation bool `yaml:"strict_validation" json:"strict_validation"`
}

func DefaultConfig() Config {
	return Config{
		DefaultPolicy:      DenyAll,
		SuperuserRole:      "admin",
		RoleHierarchy:      make(map[string][]string),
		CacheEnabled:       true,
		CacheTTL:           300, // 5 minutes
		StrictMode:         true,
		DefaultFieldPolicy: "deny",
		StrictValidation:   false,
	}
}

func (c *Config) Validate() error {
	if c.DefaultPolicy != DenyAll && c.DefaultPolicy != AllowAll {
		return &ConfigError{
			Field:   "default_policy",
			Message: fmt.Sprintf("must be '%s' or '%s'", DenyAll, AllowAll),
		}
	}

	if c.SuperuserRole == "" {
		return &ConfigError{
			Field:   "superuser_role",
			Message: "cannot be empty",
		}
	}

	if c.CacheEnabled && c.CacheTTL <= 0 {
		return &ConfigError{
			Field:   "cache_ttl",
			Message: "must be greater than 0 when cache is enabled",
		}
	}

	if c.DefaultFieldPolicy != "deny" && c.DefaultFieldPolicy != "allow" {
		return &ConfigError{
			Field:   "default_field_policy",
			Message: "must be 'deny' or 'allow'",
		}
	}

	if err := c.validateHierarchy(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateHierarchy() error {
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	var hasCycle func(role string) bool
	hasCycle = func(role string) bool {
		visited[role] = true
		recursionStack[role] = true

		for _, child := range c.RoleHierarchy[role] {
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

	for role := range c.RoleHierarchy {
		if !visited[role] {
			if hasCycle(role) {
				return &ConfigError{
					Field:   "role_hierarchy",
					Message: fmt.Sprintf("circular dependency detected involving role '%s'", role),
				}
			}
		}
	}

	return nil
}

func (c *Config) Merge(other Config) {
	if other.DefaultPolicy != "" {
		c.DefaultPolicy = other.DefaultPolicy
	}
	if other.SuperuserRole != "" {
		c.SuperuserRole = other.SuperuserRole
	}
	if len(other.RoleHierarchy) > 0 {
		if c.RoleHierarchy == nil {
			c.RoleHierarchy = make(map[string][]string)
		}
		for k, v := range other.RoleHierarchy {
			c.RoleHierarchy[k] = v
		}
	}
	if other.CacheTTL > 0 {
		c.CacheTTL = other.CacheTTL
	}
	if other.DefaultFieldPolicy != "" {
		c.DefaultFieldPolicy = other.DefaultFieldPolicy
	}
	c.CacheEnabled = other.CacheEnabled
	c.StrictMode = other.StrictMode
	c.StrictValidation = other.StrictValidation
}

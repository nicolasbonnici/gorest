package generator

import (
	"github.com/nicolasbonnici/gorest/config"
)

// AuthConfig defines which endpoints require authentication
// This is now derived from the unified config.Config
type AuthConfig struct {
	Enabled     bool                // Whether auth is enabled globally
	RequireAuth map[string][]string // resource -> HTTP methods requiring auth
}

// GetAuthConfigFromConfig creates an AuthConfig from the unified config
// Supports both new per-resource config and legacy global config
func GetAuthConfigFromConfig(cfg *config.Config) *AuthConfig {
	ac := &AuthConfig{
		Enabled:     cfg.Generate.Auth.Enabled,
		RequireAuth: make(map[string][]string),
	}

	// If auth is not enabled in config, return empty auth config
	if !cfg.Generate.Auth.Enabled {
		return ac
	}

	// NEW: Use per-resource config if available
	if len(cfg.Resources) > 0 {
		return buildAuthFromResources(cfg)
	}

	// LEGACY: Fall back to old generate.auth.endpoints config
	return buildAuthFromLegacyConfig(cfg)
}

// buildAuthFromResources creates auth config from the resources-based configuration
func buildAuthFromResources(cfg *config.Config) *AuthConfig {
	ac := &AuthConfig{
		Enabled:     cfg.Generate.Auth.Enabled,
		RequireAuth: make(map[string][]string),
	}

	// Get default methods from auth_defaults
	defaultMethods := getDefaultMethods(cfg)

	for _, resource := range cfg.Resources {
		resourceName := resource.Name
		var requiredAuthMethods []string

		// Standard HTTP methods to check
		standardMethods := []string{"GET", "POST", "PUT", "DELETE"}

		for _, method := range standardMethods {
			requireAuth := shouldRequireAuth(method, resource.Auth, defaultMethods)
			if requireAuth {
				requiredAuthMethods = append(requiredAuthMethods, method)
			}
		}

		ac.RequireAuth[resourceName] = requiredAuthMethods
	}

	return ac
}

// getDefaultMethods returns the default auth requirements for all methods
// Priority: 1. auth_defaults (top-level), 2. Secure defaults (all true)
func getDefaultMethods(cfg *config.Config) map[string]bool {
	defaults := map[string]bool{
		"GET":    true,
		"POST":   true,
		"PUT":    true,
		"DELETE": true,
	}

	// Override with auth_defaults if specified
	if cfg.AuthDefaults != nil && len(cfg.AuthDefaults) > 0 {
		for method, requireAuth := range cfg.AuthDefaults {
			defaults[method] = requireAuth
		}
	}

	return defaults
}

// shouldRequireAuth determines if a method should require auth based on:
// 1. Resource-specific config (highest priority)
// 2. Global default config (if resource not specified)
// 3. Secure default (true if neither specified)
func shouldRequireAuth(method string, resourceMethods map[string]bool, defaultMethods map[string]bool) bool {
	// Check resource-specific config first
	if requireAuth, specified := resourceMethods[method]; specified {
		return requireAuth
	}

	// Fall back to default config
	if requireAuth, exists := defaultMethods[method]; exists {
		return requireAuth
	}

	// Secure by default
	return true
}

// buildAuthFromLegacyConfig creates auth config from old generate.auth.endpoints
func buildAuthFromLegacyConfig(cfg *config.Config) *AuthConfig {
	ac := &AuthConfig{
		Enabled:     cfg.Generate.Auth.Enabled,
		RequireAuth: make(map[string][]string),
	}

	var methods []string

	// Build list of methods that require auth based on config
	if cfg.Generate.Auth.Endpoints.List {
		methods = append(methods, "GET")
	}
	if cfg.Generate.Auth.Endpoints.Get {
		// GET is already added for List, no need to duplicate
	}
	if cfg.Generate.Auth.Endpoints.Create {
		methods = append(methods, "POST")
	}
	if cfg.Generate.Auth.Endpoints.Update {
		methods = append(methods, "PUT")
	}
	if cfg.Generate.Auth.Endpoints.Delete {
		methods = append(methods, "DELETE")
	}

	// Deduplicate GET if needed
	uniqueMethods := make(map[string]bool)
	for _, m := range methods {
		uniqueMethods[m] = true
	}

	finalMethods := make([]string, 0, len(uniqueMethods))
	for m := range uniqueMethods {
		finalMethods = append(finalMethods, m)
	}

	// In legacy mode, use wildcard for all resources
	ac.RequireAuth["*"] = finalMethods
	return ac
}

// DefaultAuthConfig returns a configuration that requires auth for all CRUD operations
// This is kept for backwards compatibility with tests
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		Enabled: true,
		RequireAuth: map[string][]string{
			"users": {"GET", "POST", "PUT", "DELETE"},
			"todos": {"GET", "POST", "PUT", "DELETE"},
		},
	}
}

// NoAuthConfig returns a configuration with no authentication requirements
// This is kept for backwards compatibility with tests
func NoAuthConfig() *AuthConfig {
	return &AuthConfig{
		Enabled:     false,
		RequireAuth: map[string][]string{},
	}
}

// RequiresAuth checks if a specific resource and HTTP method requires authentication
func (c *AuthConfig) RequiresAuth(resource, method string) bool {
	// If auth is disabled globally, nothing requires auth
	if !c.Enabled {
		return false
	}

	// Try specific resource first
	methods, ok := c.RequireAuth[resource]

	// Fallback to wildcard (legacy mode)
	if !ok {
		methods, ok = c.RequireAuth["*"]
	}

	// If resource not in config, default to secure (require auth)
	if !ok {
		return true
	}

	for _, m := range methods {
		if m == method {
			return true
		}
	}
	return false
}

// SetResourceAuth sets the authentication requirements for a resource
func (c *AuthConfig) SetResourceAuth(resource string, methods []string) {
	if c.RequireAuth == nil {
		c.RequireAuth = make(map[string][]string)
	}
	c.RequireAuth[resource] = methods
}

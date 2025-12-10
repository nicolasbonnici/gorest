package codegen

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
func GetAuthConfigFromConfig(cfg *config.Config) *AuthConfig {
	ac := &AuthConfig{
		Enabled:     cfg.Codegen.Auth.Enabled,
		RequireAuth: make(map[string][]string),
	}

	// If auth is not enabled in config, return empty auth config
	if !cfg.Codegen.Auth.Enabled {
		return ac
	}

	// Build auth config from endpoints
	return buildAuthFromEndpoints(cfg)
}

// buildAuthFromEndpoints creates auth config from the endpoints configuration
func buildAuthFromEndpoints(cfg *config.Config) *AuthConfig {
	ac := &AuthConfig{
		Enabled:     cfg.Codegen.Auth.Enabled,
		RequireAuth: make(map[string][]string),
	}

	// Get default methods from auth_defaults
	defaultMethods := getDefaultMethods(cfg)

	for _, endpoint := range cfg.Codegen.Auth.Endpoints {
		endpointName := endpoint.Name
		var requiredAuthMethods []string

		// Standard HTTP methods to check
		standardMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

		for _, method := range standardMethods {
			requireAuth := shouldRequireAuth(method, endpoint, defaultMethods)
			if requireAuth {
				requiredAuthMethods = append(requiredAuthMethods, method)
			}
		}

		ac.RequireAuth[endpointName] = requiredAuthMethods
	}

	return ac
}

// getDefaultMethods returns the default auth requirements for all methods
// Priority: 1. codegen.auth.defaults, 2. Secure defaults (all true)
func getDefaultMethods(cfg *config.Config) map[string]bool {
	defaults := map[string]bool{
		"GET":    true,
		"POST":   true,
		"PUT":    true,
		"DELETE": true,
		"PATCH":  true,
	}

	// Override with codegen.auth.defaults if specified
	if cfg.Codegen.Auth.Defaults != nil && len(cfg.Codegen.Auth.Defaults) > 0 {
		for method, requireAuth := range cfg.Codegen.Auth.Defaults {
			defaults[method] = requireAuth
		}
	}

	return defaults
}

// getMethodValue retrieves the method-specific auth value from the endpoint config
func getMethodValue(endpoint config.EndpointAuthConfig, method string) (*bool, bool) {
	switch method {
	case "GET":
		return endpoint.GET, endpoint.GET != nil
	case "POST":
		return endpoint.POST, endpoint.POST != nil
	case "PUT":
		return endpoint.PUT, endpoint.PUT != nil
	case "DELETE":
		return endpoint.DELETE, endpoint.DELETE != nil
	case "PATCH":
		return endpoint.PATCH, endpoint.PATCH != nil
	}
	return nil, false
}

// shouldRequireAuth determines if a method should require auth based on:
// 1. Resource-specific config (highest priority)
// 2. Global default config (if resource not specified)
// 3. Secure default (true if neither specified)
func shouldRequireAuth(method string, endpoint config.EndpointAuthConfig, defaultMethods map[string]bool) bool {
	// Check resource-specific config first
	if value, specified := getMethodValue(endpoint, method); specified {
		return *value
	}

	// Fall back to default config
	if requireAuth, exists := defaultMethods[method]; exists {
		return requireAuth
	}

	// Secure by default
	return true
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

	// Get methods for this resource
	methods, ok := c.RequireAuth[resource]

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

package generator

import (
	"github.com/nicolasbonnici/gorest/config"
)

// AuthConfig defines which endpoints require authentication
// This is now derived from the unified config.Config
type AuthConfig struct {
	RequireAuth map[string][]string // resource -> HTTP methods requiring auth
}

// GetAuthConfigFromConfig creates an AuthConfig from the unified config
// Uses the generate.auth.endpoints settings to determine which methods require auth
func GetAuthConfigFromConfig(cfg *config.Config, resourceName string) *AuthConfig {
	ac := &AuthConfig{
		RequireAuth: make(map[string][]string),
	}

	// If auth is not enabled in config, return empty auth config
	if !cfg.Generate.Auth.Enabled {
		return ac
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

	ac.RequireAuth[resourceName] = finalMethods
	return ac
}

// DefaultAuthConfig returns a configuration that requires auth for all CRUD operations
// This is kept for backwards compatibility with tests
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
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
		RequireAuth: map[string][]string{},
	}
}

// RequiresAuth checks if a specific resource and HTTP method requires authentication
func (c *AuthConfig) RequiresAuth(resource, method string) bool {
	methods, ok := c.RequireAuth[resource]
	if !ok {
		return false
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

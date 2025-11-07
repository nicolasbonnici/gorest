package generator

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AuthConfig defines which endpoints require authentication
type AuthConfig struct {
	RequireAuth map[string][]string `json:"require_auth"` // resource -> HTTP methods requiring auth
}

// DefaultAuthConfig returns a configuration that requires auth for all CRUD operations
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		RequireAuth: map[string][]string{
			"users": {"GET", "POST", "PUT", "DELETE"},
			"todos": {"GET", "POST", "PUT", "DELETE"},
		},
	}
}

// NoAuthConfig returns a configuration with no authentication requirements
func NoAuthConfig() *AuthConfig {
	return &AuthConfig{
		RequireAuth: map[string][]string{},
	}
}

// LoadAuthConfig loads auth configuration from a JSON file
func LoadAuthConfig(path string) (*AuthConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg AuthConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// SaveAuthConfig saves auth configuration to a JSON file
func SaveAuthConfig(cfg *AuthConfig, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
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

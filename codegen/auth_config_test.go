package codegen

import (
	"testing"

	"github.com/nicolasbonnici/gorest/config"
)

// Helper function to create bool pointers
func boolPtr(b bool) *bool {
	return &b
}

// Tests for NEW format with codegen.auth.defaults and flattened resource auth

func TestGetAuthConfigFromNewAuthDefaults(t *testing.T) {
	cfg := &config.Config{
		Codegen: config.CodegenConfig{
			Auth: config.CodegenAuthConfig{
				Enabled: true,
				Defaults: map[string]bool{
					"GET":    false,
					"POST":   true,
					"PUT":    true,
					"DELETE": true,
				},
				Endpoints: []config.EndpointAuthConfig{
					{
						Name: "users",
						GET:  boolPtr(false), // Explicit in new format
					},
				},
			},
		},
	}

	authCfg := GetAuthConfigFromConfig(cfg)

	// Test explicit false in new format
	if authCfg.RequiresAuth("users", "GET") {
		t.Error("Expected GET on users to be public (false), but it requires auth")
	}

	// Test inherited from auth_defaults
	if !authCfg.RequiresAuth("users", "POST") {
		t.Error("Expected POST on users to require auth (from auth_defaults)")
	}
	if !authCfg.RequiresAuth("users", "PUT") {
		t.Error("Expected PUT on users to require auth (from auth_defaults)")
	}
	if !authCfg.RequiresAuth("users", "DELETE") {
		t.Error("Expected DELETE on users to require auth (from auth_defaults)")
	}
}

func TestGetAuthConfigNewFormatFlattenedAuth(t *testing.T) {
	cfg := &config.Config{
		Codegen: config.CodegenConfig{
			Auth: config.CodegenAuthConfig{
				Enabled: true,
				Defaults: map[string]bool{
					"GET":    true,
					"POST":   true,
					"PUT":    true,
					"DELETE": true,
				},
				Endpoints: []config.EndpointAuthConfig{
					{
						Name:   "products",
						GET:    boolPtr(false),
						POST:   boolPtr(true),
						PUT:    boolPtr(true),
						DELETE: boolPtr(true),
					},
				},
			},
		},
	}

	authCfg := GetAuthConfigFromConfig(cfg)

	if authCfg.RequiresAuth("products", "GET") {
		t.Error("Expected GET on products to be public")
	}
	if !authCfg.RequiresAuth("products", "POST") {
		t.Error("Expected POST on products to require auth")
	}
}

func TestGetAuthConfigWithDefaultPublicGet(t *testing.T) {
	cfg := &config.Config{
		Codegen: config.CodegenConfig{
			Auth: config.CodegenAuthConfig{
				Enabled: true,
				Defaults: map[string]bool{
					"GET":    false, // Default: public
					"POST":   true,
					"PUT":    true,
					"DELETE": true,
				},
				Endpoints: []config.EndpointAuthConfig{
					{
						Name: "products",
						POST: boolPtr(true), // Only override POST
					},
				},
			},
		},
	}

	authCfg := GetAuthConfigFromConfig(cfg)

	// Should inherit GET: false from default
	if authCfg.RequiresAuth("products", "GET") {
		t.Error("Expected GET on products to be public (inherited from default)")
	}

	// Should have explicit POST: true
	if !authCfg.RequiresAuth("products", "POST") {
		t.Error("Expected POST on products to require auth")
	}

	// Should inherit PUT: true from default
	if !authCfg.RequiresAuth("products", "PUT") {
		t.Error("Expected PUT on products to require auth (inherited from default)")
	}

	// Should inherit DELETE: true from default
	if !authCfg.RequiresAuth("products", "DELETE") {
		t.Error("Expected DELETE on products to require auth (inherited from default)")
	}
}

func TestGetAuthConfigSecureByDefault(t *testing.T) {
	// No defaults specified - should default to true for all
	cfg := &config.Config{
		Codegen: config.CodegenConfig{
			Auth: config.CodegenAuthConfig{
				Enabled: true,
				Endpoints: []config.EndpointAuthConfig{
					{
						Name: "orders",
						GET:  boolPtr(false), // Only specify GET
					},
				},
			},
		},
	}

	authCfg := GetAuthConfigFromConfig(cfg)

	// Explicit false
	if authCfg.RequiresAuth("orders", "GET") {
		t.Error("Expected GET on orders to be public")
	}

	// Unspecified should default to true (secure by default)
	if !authCfg.RequiresAuth("orders", "POST") {
		t.Error("Expected POST on orders to require auth (secure by default)")
	}
	if !authCfg.RequiresAuth("orders", "PUT") {
		t.Error("Expected PUT on orders to require auth (secure by default)")
	}
	if !authCfg.RequiresAuth("orders", "DELETE") {
		t.Error("Expected DELETE on orders to require auth (secure by default)")
	}
}

func TestGetAuthConfigDisabled(t *testing.T) {
	cfg := &config.Config{
		Codegen: config.CodegenConfig{
			Auth: config.CodegenAuthConfig{
				Enabled: false,
			},
		},
	}

	authCfg := GetAuthConfigFromConfig(cfg)

	// When auth is disabled globally, nothing should require auth
	if authCfg.RequiresAuth("users", "GET") {
		t.Error("Expected no auth requirement when auth is disabled globally")
	}
	if authCfg.RequiresAuth("users", "POST") {
		t.Error("Expected no auth requirement when auth is disabled globally")
	}
}

func TestGetAuthConfigResourceNotInConfig(t *testing.T) {
	cfg := &config.Config{
		Codegen: config.CodegenConfig{
			Auth: config.CodegenAuthConfig{
				Enabled: true,
				Endpoints: []config.EndpointAuthConfig{
					{
						Name: "users",
						GET:  boolPtr(false),
					},
				},
			},
		},
	}

	authCfg := GetAuthConfigFromConfig(cfg)

	// Resource not in config should default to secure (require auth)
	if !authCfg.RequiresAuth("unknown_resource", "GET") {
		t.Error("Expected unknown resource to require auth (secure by default)")
	}
	if !authCfg.RequiresAuth("unknown_resource", "POST") {
		t.Error("Expected unknown resource to require auth (secure by default)")
	}
}

func TestGetAuthConfigMultipleResources(t *testing.T) {
	cfg := &config.Config{
		Codegen: config.CodegenConfig{
			Auth: config.CodegenAuthConfig{
				Enabled: true,
				Defaults: map[string]bool{
					"GET":    false,
					"POST":   true,
					"PUT":    true,
					"DELETE": true,
				},
				Endpoints: []config.EndpointAuthConfig{
					{
						Name: "users",
						GET:  boolPtr(false),
					},
					{
						Name:   "orders",
						GET:    boolPtr(true),
						POST:   boolPtr(true),
						PUT:    boolPtr(true),
						DELETE: boolPtr(true),
					},
					{
						Name: "products",
						GET:  boolPtr(false),
					},
				},
			},
		},
	}

	authCfg := GetAuthConfigFromConfig(cfg)

	// Users: GET public
	if authCfg.RequiresAuth("users", "GET") {
		t.Error("Expected GET on users to be public")
	}
	if !authCfg.RequiresAuth("users", "POST") {
		t.Error("Expected POST on users to require auth")
	}

	// Orders: all protected
	if !authCfg.RequiresAuth("orders", "GET") {
		t.Error("Expected GET on orders to require auth")
	}
	if !authCfg.RequiresAuth("orders", "POST") {
		t.Error("Expected POST on orders to require auth")
	}

	// Products: GET public
	if authCfg.RequiresAuth("products", "GET") {
		t.Error("Expected GET on products to be public")
	}
	if !authCfg.RequiresAuth("products", "PUT") {
		t.Error("Expected PUT on products to require auth")
	}
}

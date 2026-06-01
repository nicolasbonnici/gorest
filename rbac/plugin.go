package rbac

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/plugin"
	"github.com/nicolasbonnici/gorest/rbac/migrations"
)

// Plugin implements the plugin interface for RBAC functionality
type Plugin struct {
	db     database.Database
	config Config
}

// NewPlugin creates a new RBAC plugin
func NewPlugin() plugin.Plugin {
	return &Plugin{}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "rbac"
}

// Initialize initializes the plugin with configuration
func (p *Plugin) Initialize(config map[string]interface{}) error {
	if db, ok := config["database"].(database.Database); ok {
		p.db = db
	} else {
		return fmt.Errorf("rbac plugin requires database")
	}

	// Parse RBAC configuration
	p.config = Config{
		DefaultPolicy:      DenyAll,
		SuperuserRole:      "admin",
		RoleHierarchy:      make(map[string][]string),
		CacheEnabled:       true,
		CacheTTL:           300,
		StrictMode:         false,
		DefaultFieldPolicy: "allow",
	}

	if defaultPolicy, ok := config["default_policy"].(string); ok {
		p.config.DefaultPolicy = Policy(defaultPolicy)
	}

	if superuserRole, ok := config["superuser_role"].(string); ok {
		p.config.SuperuserRole = superuserRole
	}

	if roleHierarchy, ok := config["role_hierarchy"].(map[string][]string); ok {
		p.config.RoleHierarchy = roleHierarchy
	}

	if cacheEnabled, ok := config["cache_enabled"].(bool); ok {
		p.config.CacheEnabled = cacheEnabled
	}

	if cacheTTL, ok := config["cache_ttl"].(int); ok {
		p.config.CacheTTL = cacheTTL
	}

	if strictMode, ok := config["strict_mode"].(bool); ok {
		p.config.StrictMode = strictMode
	}

	if defaultFieldPolicy, ok := config["default_field_policy"].(string); ok {
		p.config.DefaultFieldPolicy = defaultFieldPolicy
	}

	return nil
}

// SetupEndpoints registers RBAC endpoints (currently none)
func (p *Plugin) SetupEndpoints(router fiber.Router) error {
	// RBAC doesn't have its own endpoints
	return nil
}

// Handler returns the RBAC middleware (currently a pass-through)
func (p *Plugin) Handler() fiber.Handler {
	return func(c fiber.Ctx) error {
		return c.Next()
	}
}

// MigrationSource returns the migration source
func (p *Plugin) MigrationSource() interface{} {
	return migrations.GetMigrations()
}

// MigrationDependencies returns the list of plugins this plugin depends on
func (p *Plugin) MigrationDependencies() []string {
	return []string{"auth"}
}

// GetOpenAPIResources returns OpenAPI resource definitions
func (p *Plugin) GetOpenAPIResources() []plugin.OpenAPIResource {
	return []plugin.OpenAPIResource{}
}

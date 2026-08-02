package auth

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/nicolasbonnici/gorest/auth/handlers"
	authjwt "github.com/nicolasbonnici/gorest/auth/jwt"
	"github.com/nicolasbonnici/gorest/auth/middleware"
	"github.com/nicolasbonnici/gorest/auth/migrations"
	"github.com/nicolasbonnici/gorest/auth/models"
	"github.com/nicolasbonnici/gorest/auth/refresh"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/plugin"
)

// Plugin implements the plugin interface for auth functionality
type Plugin struct {
	db      database.Database
	jwt     *authjwt.Service
	refresh *refresh.Service
}

// NewPlugin creates a new auth plugin
func NewPlugin() plugin.Plugin {
	return &Plugin{}
}

// Name returns the plugin name
func (p *Plugin) Name() string {
	return "auth"
}

// Initialize initializes the plugin with configuration
func (p *Plugin) Initialize(config map[string]interface{}) error {
	if db, ok := config["database"].(database.Database); ok {
		p.db = db
	} else {
		return fmt.Errorf("auth plugin requires database")
	}

	jwtSecret, _ := config["jwt_secret"].(string)
	if jwtSecret == "" {
		return fmt.Errorf("auth plugin requires jwt_secret")
	}

	jwtTTL := 900 // default 15 minutes
	if ttl, ok := config["jwt_expiration_hours"].(int); ok {
		jwtTTL = ttl * 3600 // convert hours to seconds
	}

	p.jwt = authjwt.NewService(jwtSecret, jwtTTL)

	refreshTTL := 0 // zero selects the refresh package default
	if ttl, ok := config["refresh_ttl"].(int); ok {
		refreshTTL = ttl
	}

	p.refresh = refresh.NewService(p.db, refreshTTL)

	return nil
}

// SetupEndpoints registers authentication endpoints
func (p *Plugin) SetupEndpoints(router fiber.Router) error {
	if p.db == nil || p.jwt == nil {
		return nil
	}

	handlers.RegisterAuthRoutes(router, p.db, p.jwt, p.refresh)
	return nil
}

// Handler returns the authentication middleware
func (p *Plugin) Handler() fiber.Handler {
	if p.jwt == nil || p.db == nil {
		return func(c fiber.Ctx) error {
			return c.Next()
		}
	}
	return middleware.AuthMiddleware(p.jwt, p.db)
}

// MigrationSource returns the migration source
func (p *Plugin) MigrationSource() interface{} {
	return migrations.GetMigrations()
}

// MigrationDependencies returns the list of plugins this plugin depends on
func (p *Plugin) MigrationDependencies() []string {
	return []string{}
}

// GetOpenAPIResources returns OpenAPI resource definitions
func (p *Plugin) GetOpenAPIResources() []plugin.OpenAPIResource {
	return []plugin.OpenAPIResource{{
		Name:          "user",
		PluralName:    "users",
		BasePath:      "/users",
		Tags:          []string{"Users"},
		ResponseModel: models.User{},
		Description:   "User management and authentication",
	}}
}

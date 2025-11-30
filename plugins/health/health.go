package health

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/plugin"
)

// HealthPlugin provides a secure health check endpoint with security headers
type HealthPlugin struct {
	db      database.Database
	version string
}

func NewPlugin() plugin.Plugin {
	return &HealthPlugin{version: "dev"}
}

func (p *HealthPlugin) Name() string {
	return "health"
}

func (p *HealthPlugin) Initialize(config map[string]interface{}) error {
	if db, ok := config["database"].(database.Database); ok {
		p.db = db
	}
	if version, ok := config["__version"].(string); ok {
		p.version = version
	}
	return nil
}

// Handler returns a no-op middleware since health endpoint is set up via SetupEndpoints
func (p *HealthPlugin) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}

// SetupEndpoints implements the optional EndpointSetup interface
func (p *HealthPlugin) SetupEndpoints(app *fiber.App) error {
	app.Get("/health", p.healthCheckHandler())
	return nil
}

// healthCheckHandler creates the health check handler with security headers
func (p *HealthPlugin) healthCheckHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Block TRACE method
		if c.Method() == "TRACE" {
			return c.Status(fiber.StatusMethodNotAllowed).JSON(fiber.Map{
				"error": "Method not allowed",
			})
		}

		// Apply same security headers as security plugin for consistency
		c.Set("X-Powered-By", "GoREST/"+p.version)
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Set("Content-Security-Policy", "default-src 'self'")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// Perform health check
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()

		if p.db == nil {
			return c.JSON(fiber.Map{
				"status": "healthy",
				"database": fiber.Map{
					"status": "not_configured",
				},
			})
		}

		if err := p.db.Ping(ctx); err != nil {
			return c.Status(503).JSON(fiber.Map{
				"status": "unhealthy",
				"database": fiber.Map{
					"status": "down",
					"error":  err.Error(),
				},
			})
		}

		return c.JSON(fiber.Map{
			"status": "healthy",
			"database": fiber.Map{
				"status": "up",
			},
		})
	}
}

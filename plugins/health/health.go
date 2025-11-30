package health

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/plugin"
)

// HealthPlugin provides a health check endpoint
type HealthPlugin struct {
	db database.Database
}

func NewPlugin() plugin.Plugin {
	return &HealthPlugin{}
}

func (p *HealthPlugin) Name() string {
	return "health"
}

func (p *HealthPlugin) Initialize(config map[string]interface{}) error {
	if db, ok := config["database"].(database.Database); ok {
		p.db = db
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

// healthCheckHandler creates the health check handler
func (p *HealthPlugin) healthCheckHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
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

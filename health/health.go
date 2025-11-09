package health

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/logger"
	"github.com/nicolasbonnici/gorest/database"
)

func SetupHealthCheck(app *fiber.App, db database.Database) {
	app.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()

		// Test database connectivity
		if err := db.Ping(ctx); err != nil {
			logger.Log.Error("Database health check failed", "error", err)
			return c.Status(503).JSON(fiber.Map{
				"status": "unhealthy",
				"database": fiber.Map{
					"status": "down",
				},
			})
		}

		return c.JSON(fiber.Map{
			"status": "healthy",
			"database": fiber.Map{
				"status": "up",
			},
		})
	})
}

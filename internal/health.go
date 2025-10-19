package internal

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SetupHealthCheck registers the health check endpoint
func SetupHealthCheck(app *fiber.App, db *pgxpool.Pool) {
	app.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()

		// Test database connectivity
		if err := db.Ping(ctx); err != nil {
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
	})
}

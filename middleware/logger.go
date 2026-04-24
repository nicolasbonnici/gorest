package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/logger"
)

func Logger(environment string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		requestID := c.Locals("requestid")

		err := c.Next()

		duration := time.Since(start)
		status := c.Response().StatusCode()

		// Build full URL with query string
		url := c.OriginalURL()

		// Base log fields
		logFields := []any{
			"request_id", requestID,
			"method", c.Method(),
			"path", c.Path(),
			"url", url,
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"ip", c.IP(),
			"user_agent", c.Get("User-Agent"),
		}

		// Add error message in development environment if there's an error
		if err != nil && environment == "development" {
			logFields = append(logFields, "error", err.Error())
		}

		// Log as error if status >= 400, otherwise info
		if status >= 400 {
			logger.Log.Error("HTTP request", logFields...)
		} else {
			logger.Log.Info("HTTP request", logFields...)
		}

		return err
	}
}

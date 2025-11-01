package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/internal/logger"
)

func HTTPLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		requestID := c.Locals("requestid")

		err := c.Next()

		duration := time.Since(start)
		status := c.Response().StatusCode()

		logger.Log.Info("HTTP request",
			"request_id", requestID,
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"ip", c.IP(),
			"user_agent", c.Get("User-Agent"),
		)

		return err
	}
}

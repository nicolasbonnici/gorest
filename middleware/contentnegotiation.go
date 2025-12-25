package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ContentNegotiation validates Content-Type for mutation requests (POST, PUT, PATCH).
// Requires application/json for all mutation operations.
func ContentNegotiation() fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()
		if method == "POST" || method == "PUT" || method == "PATCH" {
			contentType := c.Get("Content-Type")
			if !strings.HasPrefix(contentType, "application/json") {
				return c.Status(fiber.StatusUnsupportedMediaType).JSON(fiber.Map{
					"error": "Content-Type must be application/json",
				})
			}
		}
		return c.Next()
	}
}

package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

// ContentNegotiation validates Content-Type for mutation requests (POST, PUT, PATCH).
// Mutations must carry application/json, or multipart/form-data for file uploads.
func ContentNegotiation() fiber.Handler {
	return func(c fiber.Ctx) error {
		method := c.Method()
		if method == "POST" || method == "PUT" || method == "PATCH" {
			contentType := c.Get("Content-Type")
			if !strings.HasPrefix(contentType, "application/json") &&
				!strings.HasPrefix(contentType, "multipart/form-data") {
				return c.Status(fiber.StatusUnsupportedMediaType).JSON(fiber.Map{
					"error": "Content-Type must be application/json or multipart/form-data",
				})
			}
		}
		return c.Next()
	}
}

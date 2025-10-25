package helpers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// RequireAuth is a decorator that wraps a handler with JWT authentication
func RequireAuth(jwtSecret string, handler fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			return c.Status(401).JSON(fiber.Map{"error": "missing token"})
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "invalid token"})
		}

		return handler(c)
	}
}

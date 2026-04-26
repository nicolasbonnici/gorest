package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	authcontext "github.com/nicolasbonnici/gorest/auth/context"
	"github.com/nicolasbonnici/gorest/auth/models"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/rbac"
)

type JWTValidator interface {
	ValidateToken(tokenString string) (string, error)
}

func AuthMiddleware(jwt JWTValidator, db database.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing authorization header",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authorization format, expected: Bearer <token>",
			})
		}

		tokenString := parts[1]
		userID, err := jwt.ValidateToken(tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired token",
			})
		}

		userUUID, err := uuid.Parse(userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "invalid user ID format",
			})
		}

		user := &models.User{ID: userUUID}
		roles, err := user.GetRoles(c.Context(), db)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to fetch user roles",
			})
		}

		c.SetUserContext(rbac.WithUser(c.Context(), userID, roles))
		authcontext.SetUserID(c, userID)

		return c.Next()
	}
}

func OptionalAuthMiddleware(jwt JWTValidator, db database.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Next()
		}

		tokenString := parts[1]
		userID, err := jwt.ValidateToken(tokenString)
		if err != nil {
			return c.Next()
		}

		userUUID, err := uuid.Parse(userID)
		if err != nil {
			return c.Next()
		}

		user := &models.User{ID: userUUID}
		roles, err := user.GetRoles(c.Context(), db)
		if err == nil {
			c.SetUserContext(rbac.WithUser(c.Context(), userID, roles))
			authcontext.SetUserID(c, userID)
		}

		return c.Next()
	}
}

package helpers

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserContextKey contextKey = "authenticated_user"

type AuthenticatedUser struct {
	UserID    string
	Email     string
	Firstname string
	Lastname  string
}

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

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			user := &AuthenticatedUser{}

			if userID, ok := claims["user_id"].(string); ok {
				user.UserID = userID
			}
			if email, ok := claims["email"].(string); ok {
				user.Email = email
			}
			if firstname, ok := claims["firstname"].(string); ok {
				user.Firstname = firstname
			}
			if lastname, ok := claims["lastname"].(string); ok {
				user.Lastname = lastname
			}

			c.Locals(UserContextKey, user)
		}

		return handler(c)
	}
}

// GetAuthenticatedUser extracts the authenticated user from Fiber context
func GetAuthenticatedUser(c *fiber.Ctx) *AuthenticatedUser {
	if user, ok := c.Locals(UserContextKey).(*AuthenticatedUser); ok {
		return user
	}
	return nil
}

// Context creates a new context with user_id from Fiber context
func Context(c *fiber.Ctx) context.Context {
	ctx := c.Context()
	if user := GetAuthenticatedUser(c); user != nil {
		return context.WithValue(ctx, "user_id", user.UserID)
	}
	return ctx
}

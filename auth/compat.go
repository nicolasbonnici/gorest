package auth

import (
	"context"

	"github.com/gofiber/fiber/v2"
	authcontext "github.com/nicolasbonnici/gorest/auth/context"
)

type AuthenticatedUser struct {
	UserID string
}

// Context is a compatibility function that returns the user context
func Context(c *fiber.Ctx) context.Context {
	return c.UserContext()
}

// GetAuthenticatedUser is a compatibility function that returns the authenticated user
func GetAuthenticatedUser(c *fiber.Ctx) *AuthenticatedUser {
	userID, ok := authcontext.GetUserID(c)
	if !ok {
		return nil
	}

	return &AuthenticatedUser{
		UserID: userID,
	}
}

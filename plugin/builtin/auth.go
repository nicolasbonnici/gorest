package builtin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/auth"
	"github.com/nicolasbonnici/gorest/plugin"
)

// AuthPlugin provides JWT authentication for routes
type AuthPlugin struct {
	jwtSecret string
}

func NewAuthPlugin() plugin.RoutePlugin {
	return &AuthPlugin{}
}

func (p *AuthPlugin) Name() string {
	return "auth"
}

func (p *AuthPlugin) Initialize(config map[string]interface{}) error {
	if secret, ok := config["jwt_secret"].(string); ok {
		p.jwtSecret = secret
	}
	return nil
}

func (p *AuthPlugin) Wrap(handler fiber.Handler) fiber.Handler {
	return auth.RequireAuth(p.jwtSecret, handler)
}

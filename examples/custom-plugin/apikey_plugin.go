package customplugin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/plugin"
)

// APIKeyPlugin provides API key based authentication for routes
type APIKeyPlugin struct {
	apiKey string
}

// NewAPIKeyPlugin creates a new API key plugin instance
func NewAPIKeyPlugin() plugin.RoutePlugin {
	return &APIKeyPlugin{}
}

func (p *APIKeyPlugin) Name() string {
	return "apikey"
}

func (p *APIKeyPlugin) Initialize(config map[string]interface{}) error {
	if key, ok := config["api_key"].(string); ok {
		p.apiKey = key
	}
	return nil
}

func (p *APIKeyPlugin) Wrap(handler fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check for API key in header
		key := c.Get("X-API-Key")

		// Also check query parameter as fallback
		if key == "" {
			key = c.Query("api_key")
		}

		// Validate API key
		if key == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "API key is required. Provide via X-API-Key header or api_key query parameter.",
			})
		}

		if key != p.apiKey {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid API key",
			})
		}

		// Store API key in context for later use
		c.Locals("api_key", key)

		// Call the wrapped handler
		return handler(c)
	}
}

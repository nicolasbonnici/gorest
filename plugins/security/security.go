package security

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/plugin"
)

type SecurityPlugin struct {
}

func NewPlugin() plugin.Plugin {
	return &SecurityPlugin{}
}

func (p *SecurityPlugin) Name() string {
	return "security"
}

func (p *SecurityPlugin) Initialize(config map[string]interface{}) error {
	return nil
}

func (p *SecurityPlugin) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Block TRACE method to prevent XST (Cross-Site Tracing) attacks
		if c.Method() == "TRACE" {
			return c.Status(fiber.StatusMethodNotAllowed).JSON(fiber.Map{
				"error": "Method not allowed",
			})
		}

		// Set security headers
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Set("Content-Security-Policy", "default-src 'self'")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		return c.Next()
	}
}

package security

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/plugin"
)

type SecurityPlugin struct {
	version string
}

func NewPlugin() plugin.Plugin {
	return &SecurityPlugin{version: "dev"}
}

func (p *SecurityPlugin) Name() string {
	return "security"
}

func (p *SecurityPlugin) Initialize(config map[string]interface{}) error {
	if version, ok := config["__version"].(string); ok {
		p.version = version
	}
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
		c.Set("X-Powered-By", "GoREST/"+p.version)
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

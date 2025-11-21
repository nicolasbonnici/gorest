package builtin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/plugin"
)

// SecurityPlugin adds security headers to all responses
type SecurityPlugin struct {
	version string
}

func NewSecurityPlugin() plugin.GlobalPlugin {
	return &SecurityPlugin{version: "dev"}
}

func (p *SecurityPlugin) SetVersion(version string) {
	p.version = version
}

func (p *SecurityPlugin) Name() string {
	return "security"
}

func (p *SecurityPlugin) Initialize(config map[string]interface{}) error {
	// No additional configuration needed
	return nil
}

func (p *SecurityPlugin) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Powered-By", "GoREST/"+p.version)
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		return c.Next()
	}
}

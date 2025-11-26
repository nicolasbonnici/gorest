package plugins

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/plugin"
)

// DummyPlugin is an example custom plugin that adds a custom header to all responses
type DummyPlugin struct {
	message string
}

func NewDummyPlugin() plugin.Plugin {
	return &DummyPlugin{
		message: "Hello from dummy plugin!",
	}
}

func (p *DummyPlugin) Name() string {
	return "dummy"
}

func (p *DummyPlugin) Initialize(config map[string]interface{}) error {
	if msg, ok := config["message"].(string); ok {
		p.message = msg
	}
	return nil
}

func (p *DummyPlugin) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Add custom header
		c.Set("X-Dummy-Plugin", p.message)

		// Log the request
		fmt.Printf("[DUMMY PLUGIN] %s %s - Message: %s\n", c.Method(), c.Path(), p.message)

		// Continue to next handler
		return c.Next()
	}
}

package customplugin

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/plugin"
)

// TimingPlugin adds request execution time headers to all responses
type TimingPlugin struct {
	enabled bool
}

// NewTimingPlugin creates a new timing plugin instance
func NewTimingPlugin() plugin.Plugin {
	return &TimingPlugin{enabled: true}
}

func (p *TimingPlugin) Name() string {
	return "timing"
}

func (p *TimingPlugin) Initialize(config map[string]interface{}) error {
	if enabled, ok := config["enabled"].(bool); ok {
		p.enabled = enabled
	}
	return nil
}

func (p *TimingPlugin) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !p.enabled {
			return c.Next()
		}

		start := time.Now()

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Add timing headers
		c.Set("X-Response-Time", fmt.Sprintf("%v", duration))
		c.Set("X-Response-Time-Ms", fmt.Sprintf("%.2f", float64(duration.Microseconds())/1000.0))

		return err
	}
}

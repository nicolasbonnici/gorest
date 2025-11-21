package builtin

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/nicolasbonnici/gorest/plugin"
)

// RateLimitPlugin provides rate limiting functionality
type RateLimitPlugin struct {
	requestsPerSecond int
	burst             int
}

func NewRateLimitPlugin() plugin.GlobalPlugin {
	return &RateLimitPlugin{
		requestsPerSecond: 100, // Default values
		burst:             200,
	}
}

func (p *RateLimitPlugin) Name() string {
	return "ratelimit"
}

func (p *RateLimitPlugin) Initialize(config map[string]interface{}) error {
	if rps, ok := config["requests_per_second"].(int); ok {
		p.requestsPerSecond = rps
	} else if rps, ok := config["requests_per_second"].(float64); ok {
		p.requestsPerSecond = int(rps)
	}

	if burst, ok := config["burst"].(int); ok {
		p.burst = burst
	} else if burst, ok := config["burst"].(float64); ok {
		p.burst = int(burst)
	}

	return nil
}

func (p *RateLimitPlugin) Handler() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        p.requestsPerSecond,
		Expiration: 1 * time.Second,
		Storage:    nil, // Use in-memory storage
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": fmt.Sprintf("Rate limit exceeded: %d requests per second", p.requestsPerSecond),
			})
		},
	})
}

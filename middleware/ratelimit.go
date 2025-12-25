package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func RateLimit(requestsPerSecond, burst int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        requestsPerSecond,
		Expiration: 1 * time.Second,
		Storage:    nil,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": fmt.Sprintf("Rate limit exceeded: %d requests per second", requestsPerSecond),
			})
		},
	})
}

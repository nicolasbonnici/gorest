package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"

	"github.com/nicolasbonnici/gorest/response"
)

// CredentialRateLimit throttles the endpoints that accept a password. The
// server-wide limiter is sized against traffic floods and measures requests per
// second, so a patient attacker guessing a few passwords a minute never trips
// it. This one counts a handful of attempts over minutes instead.
//
// The key is the client IP: the submitted email cannot be used without reading
// the body before the handler does, and an attacker controls it anyway.
func CredentialRateLimit(max int, window time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: window,
		KeyGenerator: func(c fiber.Ctx) string {
			return "credential:" + c.IP()
		},
		LimitReached: func(c fiber.Ctx) error {
			return response.SendError(c, fiber.StatusTooManyRequests, "too many authentication attempts, try again later")
		},
	})
}

package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// maxClientRequestIDLen bounds what a caller can push into every log line.
const maxClientRequestIDLen = 128

type clientRequestIDKey struct{}

// RequestID always mints the request id server-side. Fiber's middleware reuses
// any printable X-Request-ID the caller sends, which lets a client make its log
// entries collide with someone else's on purpose. A caller's id (a proxy or an
// upstream service tracing across hops) is kept apart as the client request id,
// so cross-service correlation survives without being trusted as our own.
func RequestID() fiber.Handler {
	mint := requestid.New()

	return func(c fiber.Ctx) error {
		if supplied := c.Get(fiber.HeaderXRequestID); supplied != "" {
			// The value aliases fasthttp's header buffer, which Del recycles.
			supplied = strings.Clone(supplied)
			c.Request().Header.Del(fiber.HeaderXRequestID)
			if isLoggableClientRequestID(supplied) {
				fiber.StoreInContext(c, clientRequestIDKey{}, supplied)
			}
		}
		return mint(c)
	}
}

// ClientRequestIDFromContext returns the X-Request-ID the caller sent, or "" when
// it sent none or one that was not printable ASCII within the length bound.
func ClientRequestIDFromContext(c fiber.Ctx) string {
	rid, _ := fiber.ValueFromContext[string](c, clientRequestIDKey{})
	return rid
}

func isLoggableClientRequestID(rid string) bool {
	if len(rid) > maxClientRequestIDLen {
		return false
	}
	for i := 0; i < len(rid); i++ {
		if rid[i] < 0x20 || rid[i] > 0x7e {
			return false
		}
	}
	return true
}

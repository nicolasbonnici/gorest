package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

func RequestID() fiber.Handler {
	return requestid.New()
}

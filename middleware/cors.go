package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func CORS(origins string) fiber.Handler {
	if origins == "" {
		origins = "*"
	}

	config := cors.Config{
		AllowOrigins: origins,
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
		MaxAge:       86400,
	}

	if origins != "*" {
		config.AllowCredentials = true
	}

	return cors.New(config)
}

package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
)

func CORS(origins string) fiber.Handler {
	if origins == "" {
		origins = "*"
	}

	allowOrigins := strings.Split(origins, ",")

	config := cors.Config{
		AllowOrigins: allowOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		MaxAge:       86400,
	}

	if origins != "*" {
		config.AllowCredentials = true
	}

	return cors.New(config)
}

package plugin

import (
	"github.com/gofiber/fiber/v2"
)

type Plugin interface {
	Name() string

	Initialize(config map[string]interface{}) error
}

type GlobalPlugin interface {
	Plugin

	Handler() fiber.Handler
}

type RoutePlugin interface {
	Plugin

	Wrap(handler fiber.Handler) fiber.Handler
}

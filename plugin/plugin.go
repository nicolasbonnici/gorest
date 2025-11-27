package plugin

import (
	"github.com/gofiber/fiber/v2"
)

type Plugin interface {
	Name() string

	Initialize(config map[string]interface{}) error

	Handler() fiber.Handler
}

// EndpointSetup is an optional interface for plugins that need to register endpoints.
type EndpointSetup interface {
	SetupEndpoints(app *fiber.App) error
}

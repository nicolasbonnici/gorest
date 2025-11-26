package plugin

import (
	"github.com/gofiber/fiber/v2"
)

// Plugin is the unified interface for all plugins in GoREST.
// Plugins are not auto-applied; they must be explicitly registered via app.Use() or fiber groups.
type Plugin interface {
	Name() string

	Initialize(config map[string]interface{}) error

	// Handler returns the middleware handler function.
	// This will NOT be automatically applied to the app.
	Handler() fiber.Handler
}

// EndpointSetup is an optional interface for plugins that need to register endpoints.
// Example: auth plugin registers /login and /register endpoints.
type EndpointSetup interface {
	SetupEndpoints(app *fiber.App) error
}

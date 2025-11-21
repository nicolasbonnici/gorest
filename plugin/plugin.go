package plugin

import (
	"github.com/gofiber/fiber/v2"
)

// Plugin is the base interface that all plugins must implement
type Plugin interface {
	// Name returns the unique identifier for the plugin
	Name() string

	// Initialize sets up the plugin with the provided configuration
	Initialize(config map[string]interface{}) error
}

// GlobalPlugin represents a plugin that applies to the entire application
type GlobalPlugin interface {
	Plugin

	// Handler returns the Fiber middleware handler
	Handler() fiber.Handler
}

// RoutePlugin represents a plugin that wraps individual route handlers
type RoutePlugin interface {
	Plugin

	// Wrap takes a handler and returns a wrapped version with the plugin's logic
	Wrap(handler fiber.Handler) fiber.Handler
}

package plugin

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// PluginRegistry manages all registered plugins
type PluginRegistry struct {
	globalPlugins []GlobalPlugin
	routePlugins  map[string]RoutePlugin
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		globalPlugins: make([]GlobalPlugin, 0),
		routePlugins:  make(map[string]RoutePlugin),
	}
}

// RegisterGlobal adds a global plugin to the registry
func (r *PluginRegistry) RegisterGlobal(plugin GlobalPlugin) {
	r.globalPlugins = append(r.globalPlugins, plugin)
}

// RegisterRoute adds a route-level plugin to the registry
func (r *PluginRegistry) RegisterRoute(plugin RoutePlugin) {
	r.routePlugins[plugin.Name()] = plugin
}

// ApplyGlobal applies all registered global plugins to the Fiber app in order
func (r *PluginRegistry) ApplyGlobal(app *fiber.App) error {
	for _, plugin := range r.globalPlugins {
		handler := plugin.Handler()
		if handler == nil {
			return fmt.Errorf("global plugin '%s' returned nil handler", plugin.Name())
		}
		app.Use(handler)
	}
	return nil
}

// GetRoutePlugin returns a route plugin by name
func (r *PluginRegistry) GetRoutePlugin(name string) (RoutePlugin, bool) {
	plugin, exists := r.routePlugins[name]
	return plugin, exists
}

// GetRoutePlugins returns all registered route-level plugins
func (r *PluginRegistry) GetRoutePlugins() map[string]RoutePlugin {
	return r.routePlugins
}

// GetGlobalPlugins returns all registered global plugins
func (r *PluginRegistry) GetGlobalPlugins() []GlobalPlugin {
	return r.globalPlugins
}

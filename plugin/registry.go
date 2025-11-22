package plugin

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type PluginRegistry struct {
	globalPlugins []GlobalPlugin
	routePlugins  map[string]RoutePlugin
}

func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		globalPlugins: make([]GlobalPlugin, 0),
		routePlugins:  make(map[string]RoutePlugin),
	}
}

func (r *PluginRegistry) RegisterGlobal(plugin GlobalPlugin) {
	r.globalPlugins = append(r.globalPlugins, plugin)
}

func (r *PluginRegistry) RegisterRoute(plugin RoutePlugin) {
	r.routePlugins[plugin.Name()] = plugin
}

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

func (r *PluginRegistry) GetRoutePlugin(name string) (RoutePlugin, bool) {
	plugin, exists := r.routePlugins[name]
	return plugin, exists
}

func (r *PluginRegistry) GetRoutePlugins() map[string]RoutePlugin {
	return r.routePlugins
}

func (r *PluginRegistry) GetGlobalPlugins() []GlobalPlugin {
	return r.globalPlugins
}

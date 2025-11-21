package plugin

import (
	"fmt"

	"github.com/nicolasbonnici/gorest/config"
)

// GlobalPluginFactory is a function that creates a GlobalPlugin
type GlobalPluginFactory func() GlobalPlugin

// RoutePluginFactory is a function that creates a RoutePlugin
type RoutePluginFactory func() RoutePlugin

var (
	globalPluginFactories = make(map[string]GlobalPluginFactory)
	routePluginFactories  = make(map[string]RoutePluginFactory)
)

// RegisterGlobalPluginFactory registers a factory for creating global plugins
func RegisterGlobalPluginFactory(name string, factory GlobalPluginFactory) {
	globalPluginFactories[name] = factory
}

// RegisterRoutePluginFactory registers a factory for creating route plugins
func RegisterRoutePluginFactory(name string, factory RoutePluginFactory) {
	routePluginFactories[name] = factory
}

// LoadPlugins loads and initializes plugins from configuration
func LoadPlugins(globalConfigs []config.PluginConfig, routeConfigs []config.PluginConfig, version string) (*PluginRegistry, error) {
	registry := NewPluginRegistry()

	// Load global plugins
	for _, cfg := range globalConfigs {
		if !cfg.Enabled {
			continue
		}

		plugin, err := createGlobalPlugin(cfg.Name, version, cfg.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to create global plugin '%s': %w", cfg.Name, err)
		}

		if err := plugin.Initialize(cfg.Config); err != nil {
			return nil, fmt.Errorf("failed to initialize global plugin '%s': %w", cfg.Name, err)
		}

		registry.RegisterGlobal(plugin)
	}

	// Load route plugins
	for _, cfg := range routeConfigs {
		if !cfg.Enabled {
			continue
		}

		plugin, err := createRoutePlugin(cfg.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to create route plugin '%s': %w", cfg.Name, err)
		}

		if err := plugin.Initialize(cfg.Config); err != nil {
			return nil, fmt.Errorf("failed to initialize route plugin '%s': %w", cfg.Name, err)
		}

		registry.RegisterRoute(plugin)
	}

	return registry, nil
}

// createGlobalPlugin creates a global plugin by name using registered factories
func createGlobalPlugin(name string, version string, config map[string]interface{}) (GlobalPlugin, error) {
	factory, exists := globalPluginFactories[name]
	if !exists {
		return nil, fmt.Errorf("unknown global plugin: %s", name)
	}

	plugin := factory()

	// Handle version for security plugin
	if name == "security" {
		if versionPlugin, ok := plugin.(interface{ SetVersion(string) }); ok {
			versionPlugin.SetVersion(version)
		}
	}

	return plugin, nil
}

// createRoutePlugin creates a route plugin by name using registered factories
func createRoutePlugin(name string) (RoutePlugin, error) {
	factory, exists := routePluginFactories[name]
	if !exists {
		return nil, fmt.Errorf("unknown route plugin: %s", name)
	}
	return factory(), nil
}

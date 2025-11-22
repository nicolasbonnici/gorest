package pluginloader

import (
	"fmt"

	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/plugin"
)

type GlobalPluginFactory func() plugin.GlobalPlugin

type RoutePluginFactory func() plugin.RoutePlugin

var (
	globalPluginFactories = make(map[string]GlobalPluginFactory)
	routePluginFactories  = make(map[string]RoutePluginFactory)
)

func RegisterGlobalPluginFactory(name string, factory GlobalPluginFactory) {
	globalPluginFactories[name] = factory
}

func RegisterRoutePluginFactory(name string, factory RoutePluginFactory) {
	routePluginFactories[name] = factory
}

func LoadPlugins(globalConfigs []config.PluginConfig, routeConfigs []config.PluginConfig, version string) (*plugin.PluginRegistry, error) {
	registry := plugin.NewPluginRegistry()

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

func createGlobalPlugin(name string, version string, config map[string]interface{}) (plugin.GlobalPlugin, error) {
	factory, exists := globalPluginFactories[name]
	if !exists {
		return nil, fmt.Errorf("unknown global plugin: %s", name)
	}

	p := factory()

	if name == "security" {
		if versionPlugin, ok := p.(interface{ SetVersion(string) }); ok {
			versionPlugin.SetVersion(version)
		}
	}

	return p, nil
}

func createRoutePlugin(name string) (plugin.RoutePlugin, error) {
	factory, exists := routePluginFactories[name]
	if !exists {
		return nil, fmt.Errorf("unknown route plugin: %s", name)
	}
	return factory(), nil
}

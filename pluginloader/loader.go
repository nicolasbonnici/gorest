package pluginloader

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/database"
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

		factory, exists := globalPluginFactories[cfg.Name]
		if !exists {
			return nil, fmt.Errorf("unknown global plugin '%s' - did you forget to register it?", cfg.Name)
		}

		p := factory()

		// Inject version into config for all plugins
		enrichedConfig := make(map[string]interface{})
		for k, v := range cfg.Config {
			enrichedConfig[k] = v
		}
		enrichedConfig["__version"] = version

		if err := p.Initialize(enrichedConfig); err != nil {
			return nil, fmt.Errorf("failed to initialize global plugin '%s': %w", cfg.Name, err)
		}

		registry.RegisterGlobal(p)
	}

	for _, cfg := range routeConfigs {
		if !cfg.Enabled {
			continue
		}

		factory, exists := routePluginFactories[cfg.Name]
		if !exists {
			return nil, fmt.Errorf("unknown route plugin '%s' - did you forget to register it?", cfg.Name)
		}

		p := factory()

		// Inject version into config for all plugins
		enrichedConfig := make(map[string]interface{})
		for k, v := range cfg.Config {
			enrichedConfig[k] = v
		}
		enrichedConfig["__version"] = version

		if err := p.Initialize(enrichedConfig); err != nil {
			return nil, fmt.Errorf("failed to initialize route plugin '%s': %w", cfg.Name, err)
		}

		registry.RegisterRoute(p)
	}

	return registry, nil
}

// SetupPluginEndpoints calls SetupEndpoints on all plugins that implement the EndpointSetup interface
func SetupPluginEndpoints(registry *plugin.PluginRegistry, app *fiber.App) error {
	// Check global plugins
	for _, p := range registry.GetGlobalPlugins() {
		if setupPlugin, ok := p.(plugin.EndpointSetup); ok {
			if err := setupPlugin.SetupEndpoints(app); err != nil {
				return fmt.Errorf("failed to setup endpoints for plugin '%s': %w", p.Name(), err)
			}
		}
	}

	// Check route plugins
	for _, p := range registry.GetRoutePlugins() {
		if setupPlugin, ok := p.(plugin.EndpointSetup); ok {
			if err := setupPlugin.SetupEndpoints(app); err != nil {
				return fmt.Errorf("failed to setup endpoints for plugin '%s': %w", p.Name(), err)
			}
		}
	}

	return nil
}

func InjectSharedConfig(routeConfigs []config.PluginConfig, db database.Database) []config.PluginConfig {
	enriched := make([]config.PluginConfig, len(routeConfigs))
	for i, cfg := range routeConfigs {
		enrichedCfg := make(map[string]interface{})
		for k, v := range cfg.Config {
			enrichedCfg[k] = v
		}

		enrichedCfg["database"] = db

		enriched[i] = config.PluginConfig{
			Name:    cfg.Name,
			Enabled: cfg.Enabled,
			Config:  enrichedCfg,
		}
	}
	return enriched
}

package pluginloader

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/plugin"
)

type PluginFactory func() plugin.Plugin

var pluginFactories = make(map[string]PluginFactory)

func RegisterPluginFactory(name string, factory PluginFactory) {
	pluginFactories[name] = factory
}

func LoadPlugins(configs []config.PluginConfig, version string) (*plugin.PluginRegistry, error) {
	registry := plugin.NewPluginRegistry()

	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}

		factory, exists := pluginFactories[cfg.Name]
		if !exists {
			return nil, fmt.Errorf("unknown plugin '%s' - did you forget to register it?", cfg.Name)
		}

		p := factory()

		// Inject version into config for all plugins
		enrichedConfig := make(map[string]interface{})
		for k, v := range cfg.Config {
			enrichedConfig[k] = v
		}
		enrichedConfig["__version"] = version

		if err := p.Initialize(enrichedConfig); err != nil {
			return nil, fmt.Errorf("failed to initialize plugin '%s': %w", cfg.Name, err)
		}

		registry.Register(p)
	}

	return registry, nil
}

// SetupPluginEndpoints calls SetupEndpoints on all plugins that implement the EndpointSetup interface
func SetupPluginEndpoints(registry *plugin.PluginRegistry, app *fiber.App) error {
	for _, p := range registry.GetAll() {
		if setupPlugin, ok := p.(plugin.EndpointSetup); ok {
			if err := setupPlugin.SetupEndpoints(app); err != nil {
				return fmt.Errorf("failed to setup endpoints for plugin '%s': %w", p.Name(), err)
			}
		}
	}
	return nil
}

func InjectSharedConfig(configs []config.PluginConfig, db database.Database) []config.PluginConfig {
	enriched := make([]config.PluginConfig, len(configs))
	for i, cfg := range configs {
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

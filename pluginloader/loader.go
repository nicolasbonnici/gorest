package pluginloader

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/plugin"
)

type PluginFactory func() plugin.Plugin

var (
	pluginFactories = make(map[string]PluginFactory)
	pluginMutex     sync.RWMutex
)

func RegisterPluginFactory(name string, factory PluginFactory) {
	pluginMutex.Lock()
	defer pluginMutex.Unlock()
	pluginFactories[name] = factory
}

func LoadPlugins(configs []config.PluginConfig, version string) (*plugin.PluginRegistry, error) {
	registry := plugin.NewPluginRegistry()

	pluginInfos, err := collectPluginDependencies(configs)
	if err != nil {
		return nil, err
	}

	if err := validateDependencies(pluginInfos, configs); err != nil {
		return nil, err
	}

	sortedPlugins, err := resolveInitializationOrder(pluginInfos)
	if err != nil {
		return nil, err
	}

	for _, pInfo := range sortedPlugins {
		p := pInfo.factory()

		enrichedConfig := make(map[string]interface{})
		for k, v := range pInfo.config.Config {
			enrichedConfig[k] = v
		}
		enrichedConfig[plugin.ConfigKeyVersion] = version

		if len(pInfo.dependencies) > 0 {
			deps := make(map[string]plugin.Plugin)
			for _, depName := range pInfo.dependencies {
				if depPlugin, ok := registry.Get(depName); ok {
					deps[depName] = depPlugin
				}
			}
			enrichedConfig[plugin.ConfigKeyDependencies] = deps
		}

		if err := p.Initialize(enrichedConfig); err != nil {
			return nil, fmt.Errorf("failed to initialize plugin '%s' (depends on: %v): %w",
				pInfo.name, pInfo.dependencies, err)
		}

		registry.Register(p)
	}

	return registry, nil
}

func ApplyGlobalMiddleware(registry *plugin.PluginRegistry, app *fiber.App) {
	middlewareOrder := []string{"requestid", "logger", "ratelimit", "cors", "contenttype"}
	for _, pluginName := range middlewareOrder {
		if p, ok := registry.Get(pluginName); ok {
			app.Use(p.Handler())
		}
	}
}

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

func InjectSharedConfig(configs []config.PluginConfig, db database.Database, appConfig *config.Config) []config.PluginConfig {
	enriched := make([]config.PluginConfig, len(configs))
	for i, cfg := range configs {
		enrichedCfg := make(map[string]interface{})
		for k, v := range cfg.Config {
			enrichedCfg[k] = v
		}

		enrichedCfg["database"] = db
		enrichedCfg["config"] = appConfig
		enrichedCfg["pagination_limit"] = appConfig.Pagination.DefaultLimit
		enrichedCfg["pagination_max_limit"] = appConfig.Pagination.MaxLimit
		enrichedCfg["server_scheme"] = appConfig.Server.Scheme
		enrichedCfg["server_host"] = appConfig.Server.Host
		enrichedCfg["server_port"] = appConfig.Server.Port

		if cfg.Name == "openapi" {
			projectRoot, err := findProjectRoot()
			if err == nil {
				dtosDir := filepath.Join(projectRoot, appConfig.Codegen.Output.DTOs)
				enrichedCfg["dtos_directory"] = dtosDir
			}
		}

		enriched[i] = config.PluginConfig{
			Name:    cfg.Name,
			Enabled: cfg.Enabled,
			Config:  enrichedCfg,
		}
	}
	return enriched
}

func LoadAllCommandPlugins(db database.Database, cfg *config.Config) ([]plugin.Plugin, error) {
	var commandPlugins []plugin.Plugin

	pluginConfig := map[string]interface{}{
		"database": db,
		"config":   cfg,
	}

	pluginMutex.RLock()
	defer pluginMutex.RUnlock()

	for name, factory := range pluginFactories {
		p := factory()
		if err := p.Initialize(pluginConfig); err != nil {
			return nil, fmt.Errorf("failed to initialize plugin '%s': %w", name, err)
		}

		if _, ok := p.(plugin.CommandProvider); ok {
			commandPlugins = append(commandPlugins, p)
		}
	}

	return commandPlugins, nil
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

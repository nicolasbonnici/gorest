# Custom Plugin Example

This example demonstrates how to create custom plugins for GoREST.

## Files

- **`timing_plugin.go`** - A plugin that adds request timing headers
- **`apikey_plugin.go`** - A plugin that provides API key authentication

## Plugin: Request Timing

The timing plugin adds execution time headers to all responses:

```go
type TimingPlugin struct {
    enabled bool
}

func (p *TimingPlugin) Handler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        start := time.Now()
        err := c.Next()
        duration := time.Since(start)
        c.Set("X-Response-Time", fmt.Sprintf("%v", duration))
        return err
    }
}
```

## Plugin: API Key Authentication

The API key plugin provides API key authentication middleware:

```go
type APIKeyPlugin struct {
    apiKey string
}

func (p *APIKeyPlugin) Handler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        key := c.Get("X-API-Key")
        if key == "" {
            key = c.Query("api_key")
        }
        if key != p.apiKey {
            return c.Status(401).JSON(fiber.Map{
                "error": "Invalid or missing API key"
            })
        }
        return c.Next()
    }
}
```

## Usage

### Step 1: Register Plugin Factories

In your `main.go`, register the plugin factory with the plugin loader:

```go
package main

import (
    "github.com/nicolasbonnici/gorest"
    "github.com/nicolasbonnici/gorest/pluginloader"
    "example.com/yourapp/generated/resources"
    customplugin "example.com/yourapp/plugins"
)

func init() {
    // Register custom plugins (unified interface)
    pluginloader.RegisterPluginFactory("timing", customplugin.NewTimingPlugin)
    pluginloader.RegisterPluginFactory("apikey", customplugin.NewAPIKeyPlugin)
}

func main() {
    cfg := gorest.Config{
        ConfigPath:     ".",
        RegisterRoutes: resources.RegisterGeneratedRoutes,
    }
    gorest.Start(cfg)
}
```

### Step 2: Configure in gorest.yaml

Enable and configure your plugins:

```yaml
plugins:
  - name: timing
    enabled: true
    config:
      enabled: true
  - name: apikey
    enabled: true
    config:
      api_key: "${API_KEY}"
```

### Step 3: Apply Plugins Manually

Plugins are **not automatically applied**. You must manually apply them in your application:

```go
// In your main.go or route setup
func setupRoutes(app *fiber.App, registry *plugin.PluginRegistry) {
    // Apply timing to all routes
    if timing, ok := registry.Get("timing"); ok {
        app.Use(timing.Handler())
    }

    // Create protected group with API key
    if apikey, ok := registry.Get("apikey"); ok {
        protected := app.Group("/api", apikey.Handler())
        protected.Get("/data", dataHandler)
    }

    // Public routes (no plugins applied)
    app.Get("/public", publicHandler)
}
```

The plugins will be loaded and initialized by GoREST based on your YAML configuration, but you control where they're applied.

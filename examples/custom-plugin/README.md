# Custom Plugin Example

This example demonstrates how to create custom plugins for GoREST.

## Files

- **`timing_plugin.go`** - A global plugin that adds request timing headers
- **`apikey_plugin.go`** - A route plugin that provides API key authentication

## Global Plugin: Request Timing

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

## Route Plugin: API Key Authentication

The API key plugin wraps handlers to require API key authentication:

```go
type APIKeyPlugin struct {
    apiKey string
}

func (p *APIKeyPlugin) Wrap(handler fiber.Handler) fiber.Handler {
    return func(c *fiber.Ctx) error {
        key := c.Get("X-API-Key")
        if key != p.apiKey {
            return c.Status(401).JSON(fiber.Map{
                "error": "Invalid or missing API key"
            })
        }
        return handler(c)
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
    // Register custom plugins
    pluginloader.RegisterGlobalPluginFactory("timing", customplugin.NewTimingPlugin)
    pluginloader.RegisterRoutePluginFactory("apikey", customplugin.NewAPIKeyPlugin)
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
  global:
    - name: timing
      enabled: true
      config:
        enabled: true
  route:
    - name: apikey
      enabled: true
      config:
        api_key: "${API_KEY}"
```

The plugins will be automatically loaded and initialized by GoREST based on your YAML configuration.

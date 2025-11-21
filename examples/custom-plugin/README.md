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

1. Import the plugins in your application
2. Register them with the plugin registry
3. Configure them in `gorest.yaml` or programmatically

### Programmatic Registration

```go
registry, _ := plugin.LoadPlugins(config.Plugins.Global, config.Plugins.Route, version)

// Register timing plugin
timingPlugin := customplugin.NewTimingPlugin()
timingPlugin.Initialize(map[string]interface{}{"enabled": true})
registry.RegisterGlobal(timingPlugin)

// Register API key plugin
apikeyPlugin := customplugin.NewAPIKeyPlugin()
apikeyPlugin.Initialize(map[string]interface{}{"api_key": "your-secret-key"})
registry.RegisterRoute(apikeyPlugin)
```

### YAML Configuration

To make your plugin configurable via `gorest.yaml`, register it in the plugin loader:

```yaml
plugins:
  global:
    - name: timing
      enabled: true
  route:
    - name: apikey
      enabled: true
      config:
        api_key: "${API_KEY}"
```

Then update `plugin/loader.go` to recognize your plugin names.

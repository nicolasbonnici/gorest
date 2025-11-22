# Custom Plugins Example

This directory contains custom plugins for the basic-api example.

## Dummy Plugin

The `dummy.go` file demonstrates how to create a simple custom global plugin.

### What It Does

- Adds a custom header `X-Dummy-Plugin` to all responses
- Logs each request with a custom message
- Can be configured via `gorest.yaml`

### Implementation

```go
type DummyPlugin struct {
    message string
}

func NewDummyPlugin() plugin.GlobalPlugin {
    return &DummyPlugin{
        message: "Hello from dummy plugin!",
    }
}

func (p *DummyPlugin) Handler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        c.Set("X-Dummy-Plugin", p.message)
        fmt.Printf("[DUMMY PLUGIN] %s %s\n", c.Method(), c.Path())
        return c.Next()
    }
}
```

### Registration

**Step 1:** Register the plugin factory in `main.go`:

```go
import customplugins "example.com/basic-api/plugins"

func init() {
    // Register custom plugins
    pluginloader.RegisterGlobalPluginFactory("dummy", customplugins.NewDummyPlugin)
}
```

**Step 2:** Enable and configure in `gorest.yaml`:

```yaml
plugins:
  global:
    - name: dummy
      enabled: true
      config:
        message: "Custom plugin is working!"
```

### Testing

When the server runs, every request will:
1. Include the `X-Dummy-Plugin` header in the response
2. Log a message: `[DUMMY PLUGIN] GET /users - Message: Custom plugin is working!`

You can test it with:

```bash
curl -v http://localhost:3000/health
```

Look for the `X-Dummy-Plugin` header in the response.

## Creating Your Own Plugins

To create your own plugin:

1. Create a new `.go` file in this directory
2. Implement either `plugin.GlobalPlugin` or `plugin.RoutePlugin` interface
3. Export a `NewYourPlugin()` function
4. Register it in `main.go` using `pluginloader.RegisterGlobalPluginFactory()` or `pluginloader.RegisterRoutePluginFactory()`
5. Configure it in `gorest.yaml`

### Plugin Interfaces

**GlobalPlugin** (applies to all routes):
```go
type GlobalPlugin interface {
    Name() string
    Initialize(config map[string]interface{}) error
    Handler() fiber.Handler
}
```

**RoutePlugin** (wraps specific routes):
```go
type RoutePlugin interface {
    Name() string
    Initialize(config map[string]interface{}) error
    Wrap(handler fiber.Handler) fiber.Handler
}
```

# GoREST Plugin System

A comprehensive plugin architecture for extending your REST API with modular, reusable components.

## Overview

The plugin system provides **3 distinct plugin types** for customization:

1. **Middleware Plugins** - HTTP middleware that runs on every or specific requests 
2. **Endpoint Plugins** - Create custom HTTP endpoints
3. **Command Plugins** - Add CLI commands to the code generator

## Architecture

All plugins are located in `/plugins/` and are organized by functionality:
- Each plugin has its own directory
- Plugins register themselves using `init()` functions
- Configuration is managed through `gorest.yaml`
- Plugins can access shared resources (database, config, etc.)

Soon there will be a repository avaailable

## Built-in Plugins

### Middleware Plugins

| Plugin | Description | Configuration |
|--------|-------------|---------------|
| `requestid` | Generates unique request IDs for tracing | None |
| `logger` | Structured request/response logging | None |
| `ratelimit` | Rate limiting with configurable limits | `requests_per_second`, `burst` |
| `cors` | CORS headers for cross-origin requests | `origins` (comma-separated or `*`) |
| `security` | Security headers (OWASP ASVS Level 2) | None |
| `contenttype` | Content-Type validation and negotiation | None |
| `auth` | JWT authentication middleware | `jwt_secret`, `jwt_ttl` |

### Endpoint Plugins

| Plugin | Description | Endpoints Created |
|--------|-------------|-------------------|
| `auth` | User authentication with JWT | `POST /login` |
| `health` | Health check with database ping | `GET /health` |
| `openapi` | OpenAPI documentation UI | `GET /openapi`, `GET /openapi.json` |

### Command Plugins

| Plugin | Description | Command |
|--------|-------------|---------|
| `benchmark` | API performance benchmarking | `make benchmark` |

## Plugin Interfaces

### 1. Middleware Plugin (Core)

All plugins must implement this base interface:

```go
type Plugin interface {
    Name() string
    Initialize(config map[string]interface{}) error
    Handler() fiber.Handler
}
```

**Example:**

```go
package myplugin

import (
    "github.com/gofiber/fiber/v2"
    "github.com/nicolasbonnici/gorest/plugin"
)

type MyPlugin struct {
    config string
}

func NewPlugin() plugin.Plugin {
    return &MyPlugin{}
}

func (p *MyPlugin) Name() string {
    return "myplugin"
}

func (p *MyPlugin) Initialize(cfg map[string]interface{}) error {
    if val, ok := cfg["my_config"].(string); ok {
        p.config = val
    }
    return nil
}

func (p *MyPlugin) Handler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Your middleware logic here
        c.Set("X-My-Header", p.config)
        return c.Next()
    }
}
```

### 2. Endpoint Setup Plugin (Optional)

Implement this interface to create custom HTTP endpoints:

```go
type EndpointSetup interface {
    SetupEndpoints(app *fiber.App) error
}
```

**Example:**

```go
func (p *MyPlugin) SetupEndpoints(app *fiber.App) error {
    app.Get("/custom", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{"message": "Hello from custom endpoint!"})
    })

    app.Post("/webhook", func(c *fiber.Ctx) error {
        // Handle webhook
        return c.SendStatus(200)
    })

    return nil
}
```

### 3. Command Provider Plugin (Optional)

Implement this interface to add CLI commands:

```go
type CommandProvider interface {
    Commands() []Command
}

type Command interface {
    Name() string
    Description() string
    Run(ctx *CommandContext) *CommandResult
}
```

**Example:**

```go
func (p *MyPlugin) Commands() []plugin.Command {
    return []plugin.Command{
        &MyCommand{plugin: p},
    }
}

type MyCommand struct {
    plugin *MyPlugin
}

func (c *MyCommand) Name() string {
    return "mycommand"
}

func (c *MyCommand) Description() string {
    return "Runs my custom command"
}

func (c *MyCommand) Run(ctx *plugin.CommandContext) *plugin.CommandResult {
    // Access database
    db := c.plugin.db

    // Report progress
    if ctx.ProgressCallback != nil {
        ctx.ProgressCallback("Processing...")
    }

    // Return result
    return &plugin.CommandResult{
        Success: true,
        Message: "Command completed successfully",
        FilesCreated: []string{"output.txt"},
    }
}
```

## Configuration

### Basic Configuration (gorest.yaml)

```yaml
plugins:
  - name: requestid
    enabled: true

  - name: logger
    enabled: true

  - name: ratelimit
    enabled: true
    config:
      requests_per_second: 100
      burst: 200

  - name: cors
    enabled: true
    config:
      origins: "*"

  - name: auth
    enabled: true
    config:
      jwt_secret: "${JWT_SECRET}"
      jwt_ttl: 3600
```

### Environment Variables in Configuration

Use `${VAR_NAME}` syntax to reference environment variables:

```yaml
plugins:
  - name: auth
    enabled: true
    config:
      jwt_secret: "${JWT_SECRET}"          # Required
      jwt_ttl: 900

  - name: cors
    enabled: true
    config:
      origins: "${CORS_ORIGINS}"            # e.g., "http://localhost:3000"
```

**Example .env:**
```bash
JWT_SECRET=your-secret-key-at-least-32-chars
CORS_ORIGINS=http://localhost:3000,https://myapp.com
DATABASE_URL=postgres://user:pass@localhost:5432/db
```

### Shared Configuration

All plugins automatically receive shared configuration:

```go
func (p *MyPlugin) Initialize(cfg map[string]interface{}) error {
    // Plugin-specific config
    if secret, ok := cfg["my_secret"].(string); ok {
        p.secret = secret
    }

    // Shared resources (injected automatically)
    if db, ok := cfg["database"].(database.Database); ok {
        p.db = db
    }

    if appConfig, ok := cfg["config"].(*config.Config); ok {
        p.appConfig = appConfig
    }

    if limit, ok := cfg["pagination_limit"].(int); ok {
        p.paginationLimit = limit
    }

    return nil
}
```

## Registration

### Method 1: Auto-Registration (Recommended)

Use `init()` function to register your plugin factory:

```go
package myplugin

import "github.com/nicolasbonnici/gorest/pluginloader"

func init() {
    pluginloader.RegisterPluginFactory("myplugin", NewPlugin)
}
```

### Method 2: Manual Registration

Register in your main application:

```go
import (
    "github.com/nicolasbonnici/gorest/pluginloader"
    customplugins "yourapp/plugins"
)

func init() {
    pluginloader.RegisterPluginFactory("myplugin", customplugins.NewPlugin)
}
```

## Complete Examples

### Example 1: Custom Header Plugin

**File:** `/plugins/customheader/customheader.go`

```go
package customheader

import (
    "github.com/gofiber/fiber/v2"
    "github.com/nicolasbonnici/gorest/plugin"
    "github.com/nicolasbonnici/gorest/pluginloader"
)

func init() {
    pluginloader.RegisterPluginFactory("customheader", NewPlugin)
}

type CustomHeaderPlugin struct {
    headerName  string
    headerValue string
}

func NewPlugin() plugin.Plugin {
    return &CustomHeaderPlugin{}
}

func (p *CustomHeaderPlugin) Name() string {
    return "customheader"
}

func (p *CustomHeaderPlugin) Initialize(cfg map[string]interface{}) error {
    if name, ok := cfg["header_name"].(string); ok {
        p.headerName = name
    } else {
        p.headerName = "X-Custom-Header"
    }

    if value, ok := cfg["header_value"].(string); ok {
        p.headerValue = value
    } else {
        p.headerValue = "GoREST"
    }

    return nil
}

func (p *CustomHeaderPlugin) Handler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        c.Set(p.headerName, p.headerValue)
        return c.Next()
    }
}
```

**Configuration:**
```yaml
plugins:
  - name: customheader
    enabled: true
    config:
      header_name: "X-App-Name"
      header_value: "${APP_NAME}"
```

### Example 2: Audit Log Plugin

**File:** `/plugins/auditlog/auditlog.go`

```go
package auditlog

import (
    "context"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/nicolasbonnici/gorest/database"
    "github.com/nicolasbonnici/gorest/plugin"
    "github.com/nicolasbonnici/gorest/pluginloader"
)

func init() {
    pluginloader.RegisterPluginFactory("auditlog", NewPlugin)
}

type AuditLogPlugin struct {
    db database.Database
}

func NewPlugin() plugin.Plugin {
    return &AuditLogPlugin{}
}

func (p *AuditLogPlugin) Name() string {
    return "auditlog"
}

func (p *AuditLogPlugin) Initialize(cfg map[string]interface{}) error {
    if db, ok := cfg["database"].(database.Database); ok {
        p.db = db
    }
    return nil
}

func (p *AuditLogPlugin) Handler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        start := time.Now()

        // Call next handler
        err := c.Next()

        // Log after response
        duration := time.Since(start)

        // Extract user from auth plugin
        userID := ""
        if user := c.Locals("authenticated_user"); user != nil {
            // Type assertion based on your auth plugin
            userID = "user_id_from_context"
        }

        // Save to database
        if p.db != nil {
            query := `INSERT INTO audit_log (user_id, method, path, status, duration_ms, created_at)
                     VALUES ($1, $2, $3, $4, $5, $6)`
            p.db.Exec(context.Background(), query,
                userID,
                c.Method(),
                c.Path(),
                c.Response().StatusCode(),
                duration.Milliseconds(),
                time.Now(),
            )
        }

        return err
    }
}
```

### Example 3: Webhook Endpoint Plugin

**File:** `/plugins/webhook/webhook.go`

```go
package webhook

import (
    "github.com/gofiber/fiber/v2"
    "github.com/nicolasbonnici/gorest/plugin"
    "github.com/nicolasbonnici/gorest/pluginloader"
)

func init() {
    pluginloader.RegisterPluginFactory("webhook", NewPlugin)
}

type WebhookPlugin struct {
    secret string
}

func NewPlugin() plugin.Plugin {
    return &WebhookPlugin{}
}

func (p *WebhookPlugin) Name() string {
    return "webhook"
}

func (p *WebhookPlugin) Initialize(cfg map[string]interface{}) error {
    if secret, ok := cfg["webhook_secret"].(string); ok {
        p.secret = secret
    }
    return nil
}

func (p *WebhookPlugin) Handler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        return c.Next()
    }
}

// SetupEndpoints creates the webhook endpoint
func (p *WebhookPlugin) SetupEndpoints(app *fiber.App) error {
    app.Post("/webhook", func(c *fiber.Ctx) error {
        // Verify signature
        signature := c.Get("X-Webhook-Signature")
        if signature != p.secret {
            return c.Status(401).JSON(fiber.Map{"error": "invalid signature"})
        }

        // Process webhook payload
        var payload map[string]interface{}
        if err := c.BodyParser(&payload); err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
        }

        // Handle webhook (e.g., send to queue, process async)
        // ...

        return c.JSON(fiber.Map{"status": "received"})
    })

    return nil
}
```

**Configuration:**
```yaml
plugins:
  - name: webhook
    enabled: true
    config:
      webhook_secret: "${WEBHOOK_SECRET}"
```

## Middleware Execution Order

Plugins are applied in a specific order for optimal security and functionality:

```
1. requestid    - Generate unique request ID
2. logger       - Log incoming request
3. ratelimit    - Check rate limits
4. cors         - Handle CORS preflight
5. security     - Add security headers
6. contenttype  - Validate Content-Type
7. auth         - JWT authentication (optional, route-specific)
```

**Custom Order:**

To customize the order, modify `/pluginloader/loader.go`:

```go
func ApplyGlobalMiddleware(registry *plugin.PluginRegistry, app *fiber.App) {
    middlewareOrder := []string{
        "requestid",
        "logger",
        "myplugin",      // Your custom plugin
        "ratelimit",
        "cors",
        "security",
        "contenttype",
    }
    // ...
}
```

## Plugin Configuration Reference

### Auth Plugin

```yaml
plugins:
  - name: auth
    enabled: true
    config:
      jwt_secret: "${JWT_SECRET}"    # Required: Secret key for JWT signing (min 32 chars)
      jwt_ttl: 900                   # Token expiry in seconds (default: 900 = 15 min)
```

**Creates endpoints:**
- `POST /login` - Authenticate with email/password, returns JWT token

### Rate Limit Plugin

```yaml
plugins:
  - name: ratelimit
    enabled: true
    config:
      requests_per_second: 100       # Max requests per second per IP
      burst: 200                     # Max burst size
```

### CORS Plugin

```yaml
plugins:
  - name: cors
    enabled: true
    config:
      origins: "*"                   # Allowed origins (use "*" for all, or comma-separated URLs)
```

### OpenAPI Plugin

```yaml
plugins:
  - name: openapi
    enabled: true
```

**Creates endpoints:**
- `GET /openapi` - Scalar UI for API documentation
- `GET /openapi.json` - Dynamic OpenAPI 3.0 specification

### Benchmark Plugin

```yaml
plugins:
  - name: benchmark
    enabled: true  # Enable for CLI command
```

**CLI Command:**
```bash
gorest benchmark
```

Runs performance benchmarks with varying concurrency and data sizes.

## Testing Plugins

### Unit Testing

```go
func TestMyPlugin_Handler(t *testing.T) {
    // Initialize plugin
    p := NewPlugin().(*MyPlugin)
    cfg := map[string]interface{}{
        "my_config": "test_value",
    }
    p.Initialize(cfg)

    // Create test app
    app := fiber.New()
    app.Use(p.Handler())
    app.Get("/test", func(c *fiber.Ctx) error {
        return c.SendString("OK")
    })

    // Test request
    req := httptest.NewRequest("GET", "/test", nil)
    resp, _ := app.Test(req)

    // Verify
    if resp.StatusCode != 200 {
        t.Errorf("Expected 200, got %d", resp.StatusCode)
    }

    if resp.Header.Get("X-My-Header") != "test_value" {
        t.Error("Header not set correctly")
    }
}
```

### Integration Testing

```go
func TestPluginIntegration(t *testing.T) {
    // Load config
    cfg := &config.Config{
        Plugins: []config.PluginConfig{
            {
                Name:    "myplugin",
                Enabled: true,
                Config: map[string]interface{}{
                    "my_config": "value",
                },
            },
        },
    }

    // Load plugins
    registry, err := pluginloader.LoadPlugins(cfg.Plugins, "1.0.0")
    if err != nil {
        t.Fatal(err)
    }

    // Test plugin exists
    p, exists := registry.Get("myplugin")
    if !exists {
        t.Fatal("Plugin not loaded")
    }

    if p.Name() != "myplugin" {
        t.Errorf("Expected 'myplugin', got '%s'", p.Name())
    }
}
```

## Best Practices

### ✅ Do

- **Use `init()` for registration** - Automatic, clean, and idiomatic
- **Validate configuration** - Return errors from `Initialize()` if config is invalid
- **Use environment variables** - For secrets and environment-specific config
- **Implement `EndpointSetup`** - When creating custom endpoints
- **Keep plugins focused** - One responsibility per plugin
- **Document your plugin** - Add README.md in plugin directory

### ❌ Don't

- **Don't block in Handler()** - Keep middleware fast, use goroutines for async work
- **Don't panic** - Return errors instead
- **Don't hardcode values** - Use configuration instead
- **Don't skip initialization** - Always validate config in `Initialize()`
- **Don't modify request body** - Unless that's the plugin's purpose

## Common Patterns

### Pattern 1: Database Access

```go
type MyPlugin struct {
    db database.Database
}

func (p *MyPlugin) Initialize(cfg map[string]interface{}) error {
    if db, ok := cfg["database"].(database.Database); ok {
        p.db = db
    }
    return nil
}

func (p *MyPlugin) Handler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Use database
        if p.db != nil {
            // Query database
        }
        return c.Next()
    }
}
```

### Pattern 2: Context Injection

```go
func (p *MyPlugin) Handler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Add to Fiber Locals (available in route handlers)
        c.Locals("my_data", someValue)

        // Or modify the Go context
        ctx := context.WithValue(c.Context(), "my_key", someValue)
        c.SetUserContext(ctx)

        return c.Next()
    }
}
```

### Pattern 3: Conditional Middleware

```go
func (p *MyPlugin) Handler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Skip for certain paths
        if c.Path() == "/health" {
            return c.Next()
        }

        // Apply logic
        // ...

        return c.Next()
    }
}
```

## Project Structure

```
yourapp/
├── plugins/
│   ├── myplugin/
│   │   ├── myplugin.go      # Plugin implementation
│   │   ├── myplugin_test.go # Unit tests
│   │   └── README.md        # Plugin documentation
│   └── ...
├── main.go                  # App entry point
├── gorest.yaml             # Configuration
└── .env                    # Environment variables
```

## See Also

- `/plugins/` - Built-in plugin implementations
- `/plugin/plugin.go` - Plugin interfaces
- `/pluginloader/loader.go` - Plugin loader implementation
- `/examples/basic-api/plugins/` - Custom plugin examples
- [HOOKS.md](HOOKS.md) - Business logic hooks system

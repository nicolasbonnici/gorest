# GoREST

🚀 **GoREST** is a Go library for building type-safe REST APIs in Go from your database schema.

**Use GoREST as:**
- 📦 **A Go library** - Import packages for CRUD, filters, pagination, auth
- 🛠️ **A code generator** - Scaffold complete REST APIs from your database

## ✨ Features

- 🔎 Auto-discovery of tables, relations, columns & types
- 🛠 Scaffold REST endpoints for each table
- ⚡ Offer Type-safe generic CRUD operations with hooks system
- 🔐 Full DTO support with field-level control (`dto` tags)
- 🔑 JWT authentication with context-aware plugins
- 🎭 Hook layer to add your business logic onto your API resources
- 🧩 Modular plugin system for easy customization
- 🌐 JSON-LD support with semantic web context (@context, @type, @id)
- 🔗 Automatic relation to IRI conversion (e.g., `/users/{uuid}`)
- 🔍 Advanced filtering & ordering 
- 📄 Page based pagination with Hydra collections
- 👨🏻‍💻 DAL for PostgreSQL, MySQL and SQLite engines
- 🛡️ Production grade errors and processes management
- 🐳 Docker support with multi-database testing
- 🧪 Full test coverage with automated testing
- 💚 Health check endpoint (`/health`)
- 📜 OpenAPI 3 spec generation

---

## 🚀 Quick Start

### 1. Create Your Project
```bash
mkdir my-api && cd my-api
go mod init github.com/yourusername/my-api
go get github.com/nicolasbonnici/gorest@latest
```

### 2. Configure Your API

Create `gorest.yaml` in your project root:

```yaml
codegen:
  output:
    models: "generated/models"
    resources: "generated/resources"
    dtos: "generated/dtos"
    openapi: "generated/openapi"
    config: "generated/config"
  auth:
    enabled: true
    # Secure by default - all methods require auth unless overridden
    defaults:
      GET: true
      POST: true
      PUT: true
      DELETE: true
    endpoints:
      - name: users
        GET: true
        POST: true
        PUT: true
        DELETE: true
      - name: posts
        GET: false    # Public read
        POST: true
        PUT: true
        DELETE: true

server:
  port: 3000
  environment: "development"

database:
  url: "${DATABASE_URL}"

pagination:
  default_limit: 10
  max_limit: 1000

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
  - name: security
    enabled: true
  - name: contenttype
    enabled: true
  - name: auth
    enabled: true
    config:
      jwt_secret: "${JWT_SECRET}"
      jwt_ttl: 900
```

Set required environment variables:
```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/mydb?sslmode=require"
export JWT_SECRET=$(openssl rand -base64 32)
```

### 3. Generate Code from Your Database
```bash
go run github.com/nicolasbonnici/gorest/cmd/codegen@latest all
# Or run individual steps:
# go run github.com/nicolasbonnici/gorest/cmd/codegen@latest models
# go run github.com/nicolasbonnici/gorest/cmd/codegen@latest resources
# go run github.com/nicolasbonnici/gorest/cmd/codegen@latest openapi
```

### 4. Create Your Main Application
```go
package main

import (
    "github.com/yourusername/my-api/generated/resources"
    "github.com/nicolasbonnici/gorest"
)

func main() {
    cfg := gorest.Config{
        ConfigPath:     ".",
        RegisterRoutes: resources.RegisterGeneratedRoutes,
    }
    gorest.Start(cfg)
}
```

### 5. Run Your API
```bash
go run main.go
```

Your API is now running at: **http://localhost:3000/**
- 📚 API specs: **http://localhost:3000/openapi** (JSON format **http://localhost:3000/openapi.json**)
- 💚 Health: **http://localhost:3000/health**

---

## ⚙️ Configuration

GoREST uses `gorest.yaml` for all configuration. The file has four main sections:

### Code Generation (`codegen`)

Controls how code is generated from your database:

```yaml
codegen:
  output:
    models: "generated/models"       # Where to generate models
    resources: "generated/resources" # Where to generate API handlers
    dtos: "generated/dtos"          # Where to generate DTOs
    openapi: "generated/openapi"    # Where to generate OpenAPI
    config: "generated/config"      # Where to generate config files

  auth:
    enabled: true

    # Defaults - applied to all endpoints unless overridden
    # Secure by default: all methods require auth
    defaults:
      GET: true
      POST: true
      PUT: true
      DELETE: true

    endpoints:
      - name: posts
        GET: false  # Public read access
```

### Runtime Configuration (`server`, `database`, `pagination`)

Basic server settings:

```yaml
server:
  port: 3000
  environment: "development"

database:
  url: "${DATABASE_URL}"

pagination:
  default_limit: 10
  max_limit: 1000
```

### Plugin Configuration

All middleware and features are configured through plugins. Plugins are loaded in the order specified and must be manually applied to routes or route groups:

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
  - name: security
    enabled: true
  - name: contenttype
    enabled: true
  - name: health
    enabled: true
  - name: auth
    enabled: true
    config:
      jwt_secret: "${JWT_SECRET}"
      jwt_ttl: 900
```

### Environment-Specific Overrides

Create `gorest.{environment}.yaml` files to override base config:

**gorest.production.yaml**:
```yaml
server:
  environment: "production"

plugins:
  - name: cors
    enabled: true
    config:
      origins: "https://app.example.com"
  - name: ratelimit
    enabled: true
    config:
      requests_per_second: 50
      burst: 100
```

Load with `ENVIRONMENT` variable:
```bash
export ENVIRONMENT=production
```

### Template

See [`gorest.yaml.example`](gorest.yaml.example) for a complete documented template.

---

## 📦 Using GoREST as a Library

Import GoREST packages directly in your Go projects:

```bash
go get github.com/nicolasbonnici/gorest@latest
```

### Available Packages

| Package | Description |
|---------|-------------|
| `crud` | Type-safe CRUD operations with hooks |
| `database` | Multi-database abstraction |
| `filter` | Query filtering & ordering |
| `serializer` | JSON-LD response serialization |
| `hooks` | Lifecycle hooks for business logic |
| `plugin` | Plugin interfaces (core only - no implementations) |
| `pluginloader` | Plugin factory registration system |
| `plugins/*` | Built-in plugin implementations (separate from core) |
| `pagination` | Hydra-compliant pagination |
| `response` | HTTP response helpers |

### Example

```go
import (
    "github.com/gofiber/fiber/v2"
    "github.com/nicolasbonnici/gorest/database"
    "github.com/nicolasbonnici/gorest/crud"
    auth "github.com/nicolasbonnici/gorest/plugins/auth"
)

type User struct {
    ID    string `json:"id" db:"id"`
    Email string `json:"email" db:"email"`
}

func (User) TableName() string { return "users" }

func main() {
    db, _ := database.Open("postgres", "postgres://...")
    defer db.Close()

    userCRUD := crud.New[User](db)
    app := fiber.New()

    app.Get("/users", auth.RequireAuth("secret", func(c *fiber.Ctx) error {
        ctx := auth.Context(c)
        result, _ := userCRUD.GetAllPaginated(ctx, crud.PaginationOptions{Limit: 10})
        return c.JSON(result.Items)
    }))

    app.Listen(":3000")
}
```

📚 [Full API documentation on pkg.go.dev](https://pkg.go.dev/github.com/nicolasbonnici/gorest)

---

## 🧩 Plugin System

GoREST uses a modular unified plugin system for API customization. All plugins implement the same `Plugin` interface and return a Fiber middleware handler. Plugins are **not automatically applied** - you must manually use them with `app.Use()` or apply them to specific route groups.

### Built-in Plugins

- **requestid** - Adds unique request ID tracking
- **logger** - HTTP request/response logging
- **ratelimit** - Per-IP rate limiting
- **cors** - Cross-Origin Resource Sharing
- **security** - Security headers (X-Frame-Options, CSP, HSTS, etc.) and TRACE method blocking
- **contenttype** - Validates Content-Type for mutations
- **health** - Secure health check endpoint with database connectivity monitoring
- **auth** - JWT authentication for protected routes

### Configuration

Plugins are configured in `gorest.yaml` as a flat list:

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
  - name: auth
    enabled: true
    config:
      jwt_secret: "${JWT_SECRET}"
      jwt_ttl: 900
```

### Creating Custom Plugins

All plugins implement a single unified `Plugin` interface:

```go
package myplugin

import (
    "github.com/gofiber/fiber/v2"
    "github.com/nicolasbonnici/gorest/plugin"
)

type CustomPlugin struct {
    headerValue string
}

func NewCustomPlugin() plugin.Plugin {
    return &CustomPlugin{}
}

func (p *CustomPlugin) Name() string {
    return "custom"
}

func (p *CustomPlugin) Initialize(config map[string]interface{}) error {
    if val, ok := config["header_value"].(string); ok {
        p.headerValue = val
    }
    return nil
}

func (p *CustomPlugin) Handler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Add custom header to all requests
        c.Set("X-Custom-Header", p.headerValue)
        return c.Next()
    }
}
```

#### Auth/Validation Plugin Example

For plugins that need to protect or validate routes:

```go
type APIKeyPlugin struct {
    apiKey string
}

func NewAPIKeyPlugin() plugin.Plugin {
    return &APIKeyPlugin{}
}

func (p *APIKeyPlugin) Name() string {
    return "apikey"
}

func (p *APIKeyPlugin) Initialize(config map[string]interface{}) error {
    if key, ok := config["api_key"].(string); ok {
        p.apiKey = key
    }
    return nil
}

func (p *APIKeyPlugin) Handler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        key := c.Get("X-API-Key")
        if key != p.apiKey {
            return c.Status(401).JSON(fiber.Map{"error": "Invalid API key"})
        }
        return c.Next()
    }
}
```

### Registering Custom Plugins

Register plugin factories in your main application using `init()`:

```go
package main

import (
    "github.com/nicolasbonnici/gorest"
    "github.com/nicolasbonnici/gorest/pluginloader"

    // Import built-in plugins you want to use
    authplugin "github.com/nicolasbonnici/gorest/plugins/auth"
    loggerplugin "github.com/nicolasbonnici/gorest/plugins/logger"
    corsplugin "github.com/nicolasbonnici/gorest/plugins/cors"

    // Import your custom plugins
    customplugins "yourapp/plugins"
)

func init() {
    // Register built-in plugins
    pluginloader.RegisterPluginFactory("auth", authplugin.NewPlugin)
    pluginloader.RegisterPluginFactory("logger", loggerplugin.NewPlugin)
    pluginloader.RegisterPluginFactory("cors", corsplugin.NewPlugin)

    // Register custom plugins
    pluginloader.RegisterPluginFactory("custom", customplugins.NewCustomPlugin)
    pluginloader.RegisterPluginFactory("apikey", customplugins.NewAPIKeyPlugin)
}

func main() {
    cfg := gorest.Config{
        ConfigPath:     ".",
        RegisterRoutes: resources.RegisterGeneratedRoutes,
    }
    gorest.Start(cfg)
}
```

### Applying Plugins to Routes

Plugins are **not automatically applied**. You must manually apply them to your application or specific route groups:

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/nicolasbonnici/gorest/pluginloader"
)

func main() {
    app := fiber.New()

    // Load plugins from config
    registry, _ := pluginloader.LoadPlugins(config.Plugins, version)

    // Apply global plugins to all routes
    if requestid, ok := registry.Get("requestid"); ok {
        app.Use(requestid.Handler())
    }
    if logger, ok := registry.Get("logger"); ok {
        app.Use(logger.Handler())
    }
    if cors, ok := registry.Get("cors"); ok {
        app.Use(cors.Handler())
    }

    // Create a protected route group with auth plugin
    if authPlugin, ok := registry.Get("auth"); ok {
        protected := app.Group("/api", authPlugin.Handler())

        // Register protected routes
        protected.Get("/todos", todoHandler)
        protected.Post("/todos", createTodoHandler)
    }

    // Public routes (no auth)
    app.Post("/auth/login", loginHandler)
    app.Post("/auth/register", registerHandler)

    app.Listen(":3000")
}
```

Plugins are configured in `gorest.yaml` and loaded using `pluginloader.LoadPlugins()`. The registration order in YAML doesn't matter - you control the application order in your code.

---

## 🪝 Hooks System

Customize business logic without modifying generated code:

```go
type TodoHooks struct {
    hooks.NoOpHooks[models.Todo]
}

func (h *TodoHooks) StateProcessor(ctx context.Context, operation hooks.Operation, id any, todo *models.Todo) error {
    if operation == hooks.OperationCreate {
        if todo.Title == "" || len(todo.Title) < 3 {
            return fmt.Errorf("title must be at least 3 characters")
        }

        // Auto-populate user_id from JWT context (server-side)
        if userID := ctx.Value("user_id"); userID != nil {
            todo.UserId = &(userID.(string))
        }
    }
    return nil
}
```

See [HOOKS.md](HOOKS.md) for complete documentation.

---

## 🔐 DTOs & Field Control

Control field visibility with the `dto` struct tag:

```go
type Todo struct {
    Id        string     `json:"id" db:"id"`
    UserId    *string    `json:"user_id" db:"user_id" dto:"read"`  // Read-only
    Title     string     `json:"title" db:"title"`
    Content   string     `json:"content" db:"content"`
    CreatedAt *time.Time `json:"created_at" db:"created_at"`
}
```

Generated DTOs:
- **CreateDTO** (POST) - Excludes `id`, `user_id`, `created_at`
- **UpdateDTO** (PUT) - Excludes `id`, `user_id`, `created_at`  
- **ResponseDTO** (GET) - Includes all fields marked with `dto:"read"`

**Tags:**
- `dto:"-"` - Exclude from all DTOs
- `dto:"read"` - Only in responses
- `dto:"write"` - Only in create/update
- `dto:"read,write"` - Include in all (default)

---

## 🔍 Filtering & Ordering

### Filters

```bash
# Equality
GET /todos?status=active

# Multiple values (OR)
GET /todos?status[]=active&status[]=pending

# Comparison operators
GET /todos?priority[gte]=5
GET /todos?priority[lt]=10

# Text search
GET /todos?title[like]=meeting
GET /todos?title[ilike]=MEETING  # Case-insensitive

# Combine filters (AND)
GET /todos?status=active&priority[gte]=7
```

### Ordering

```bash
# Single field
GET /todos?order[created_at]=desc

# Multiple fields  
GET /todos?order[priority]=desc&order[created_at]=asc
```

---

## 🌐 JSON-LD Support

Automatic semantic web support with content negotiation:

```bash
# Regular JSON
curl -H "Accept: application/json" http://localhost:3000/todos/123

# JSON-LD
curl -H "Accept: application/ld+json" http://localhost:3000/todos/123
```

JSON-LD response:
```json
{
  "@context": "https://schema.org/",
  "@type": "TodoDTO",
  "@id": "/todos/abc-123",
  "id": "abc-123",
  "user_id": "/users/def-456",
  "title": "Buy groceries"
}
```

Foreign keys automatically convert to IRIs (`user_id` → `/users/def-456`).

---

## 📂 Project Structure

### Generated Project
```
my-api/
├── gorest.yaml              # Configuration
├── main.go                   # Your application
├── generated/
│   ├── models/              # DB models
│   ├── resources/           # REST handlers
│   ├── dtos/               # Data transfer objects
│   └── openapi/            # OpenAPI schema
```

### GoREST Library
```
gorest/
├── crud/                    # Generic CRUD
├── database/               # Multi-DB abstraction
│   ├── postgres/
│   ├── mysql/
│   └── sqlite/
├── filter/                 # Query filtering
├── serializer/             # JSON-LD serialization
├── codegen/                # Code generation
├── health/                 # Health check endpoint
├── hooks/                  # Lifecycle hooks
├── logger/                 # Logging utilities
├── middleware/             # HTTP middleware
├── pagination/             # Hydra pagination
├── plugin/                 # Plugin interfaces (core)
├── pluginloader/           # Plugin factory & loading system
├── plugins/                # Built-in plugin implementations
│   ├── auth/              # JWT authentication
│   ├── contenttype/       # Content-Type validation
│   ├── cors/              # CORS handling
│   ├── logger/            # HTTP logging
│   ├── ratelimit/         # Rate limiting
│   ├── requestid/         # Request ID tracking
│   └── security/          # Security headers
├── response/               # HTTP response helpers
└── cmd/                    # CLI tool
    └── codegen/           # Unified code generator (models, resources, DTOs, OpenAPI)
```

---

## 🛠 Development Commands

```bash
# Code Generation
make codegen          # Run all code generation (models, resources, DTOs, OpenAPI)
make codegen-models   # Generate models only
make codegen-resources # Generate resources & DTOs only
make codegen-openapi  # Generate OpenAPI schema only
make generate         # Alias for codegen

# Testing
make test-up          # Start test databases (PostgreSQL, MySQL)
make test-schema      # Load test schema
make test-generate    # Generate code for tests
make test             # Run all tests
make test-coverage    # Run tests with coverage report

# Benchmarking
make benchmark        # Run API performance benchmarks
```

---

## 🚀 Production Deployment

### Docker

```yaml
services:
  api:
    build: .
    ports: ["3000:3000"]
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - JWT_SECRET=${JWT_SECRET}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/health"]
      interval: 30s
```

### Nginx Reverse Proxy

```nginx
upstream gorest {
    server localhost:3000;
}

server {
    listen 443 ssl;
    server_name api.example.com;
    
    location / {
        proxy_pass http://gorest;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Kubernetes Health Checks

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 3000
  initialDelaySeconds: 10
  periodSeconds: 30
```

---

## 🔒 Security

- **Passwords**: bcrypt hashing with automatic salts
- **JWT**: 32+ character secrets required
- **CORS**: Configurable origins
- **Rate Limiting**: Configurable per-IP limits
- **SQL Injection**: Parameterized queries
- **Input Validation**: go-playground/validator support

---

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

Quick start:
```bash
git clone https://github.com/YOUR_USERNAME/gorest.git
cd gorest
make test-up && make test-schema && make test
```

---

## 📋 Changelog

See [CHANGELOG.md](CHANGELOG.md) for release history.

---

## 📜 License

MIT – free to use in your projects 🚀

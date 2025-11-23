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
generate:
  output:
    models: "generated/models"
    resources: "generated/resources"
    dtos: "generated/dtos"
    openapi: "generated/openapi"
    config: "generated/config"
  auth:
    enabled: true
    endpoints:
      list: true
      get: true
      create: true
      update: true
      delete: true

server:
  port: 3000
  environment: "development"

database:
  url: "${DATABASE_URL}"

pagination:
  default_limit: 10
  max_limit: 1000

plugins:
  global:
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
  route:
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
go run github.com/nicolasbonnici/gorest/cmd/modelgen@latest
go run github.com/nicolasbonnici/gorest/cmd/resourcegen@latest
go run github.com/nicolasbonnici/gorest/cmd/openapigen@latest
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

GoREST uses `gorest.yaml` for all configuration. The file has three main sections:

### Code Generation (`generate`)

Controls how code is generated from your database:

```yaml
generate:
  output:
    models: "generated/models"       # Where to generate models
    resources: "generated/resources" # Where to generate API handlers
    dtos: "generated/dtos"          # Where to generate DTOs
    openapi: "generated/openapi"    # Where to generate OpenAPI
    config: "generated/config"      # Where to generate config files

  auth:
    enabled: true                   # Require auth by default?
    endpoints:
      list: true                    # Require auth for GET /resource
      get: true                     # Require auth for GET /resource/:id
      create: true                  # Require auth for POST /resource
      update: true                  # Require auth for PUT /resource/:id
      delete: true                  # Require auth for DELETE /resource/:id
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

All middleware and features are configured through plugins:

```yaml
plugins:
  global:                   # Plugins applied to all routes
    - name: requestid
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
    - name: logger
      enabled: true
  route:                    # Plugins applied to specific routes
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
  global:
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
| `formatter` | JSON-LD response formatting |
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

GoREST uses a modular plugin system for API customization. Plugins can be global (applied to all routes) or route-level (applied to specific handlers).

### Built-in Plugins

**Global Plugins:**
- **requestid** - Adds unique request ID tracking
- **ratelimit** - Per-IP rate limiting
- **cors** - Cross-Origin Resource Sharing
- **security** - Security headers (X-Frame-Options, CSP, etc.)
- **contenttype** - Validates Content-Type for mutations
- **logger** - HTTP request/response logging

**Route Plugins:**
- **auth** - JWT authentication for protected routes

### Configuration

Plugins are configured in `gorest.yaml`:

```yaml
plugins:
  global:
    - name: ratelimit
      enabled: true
      config:
        requests_per_second: 100
        burst: 200
  route:
    - name: auth
      enabled: true
      config:
        jwt_secret: "${JWT_SECRET}"
```

### Creating Custom Plugins

#### Global Plugin Example

```go
package myplugin

import (
    "github.com/gofiber/fiber/v2"
    "github.com/nicolasbonnici/gorest/plugin"
)

type CustomPlugin struct {
    config map[string]interface{}
}

func NewCustomPlugin() plugin.GlobalPlugin {
    return &CustomPlugin{}
}

func (p *CustomPlugin) Name() string {
    return "custom"
}

func (p *CustomPlugin) Initialize(config map[string]interface{}) error {
    p.config = config
    return nil
}

func (p *CustomPlugin) Handler() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Your plugin logic here
        c.Set("X-Custom-Header", "value")
        return c.Next()
    }
}
```

#### Route Plugin Example

```go
type CustomAuthPlugin struct {
    apiKey string
}

func (p *CustomAuthPlugin) Name() string {
    return "apikey"
}

func (p *CustomAuthPlugin) Initialize(config map[string]interface{}) error {
    if key, ok := config["api_key"].(string); ok {
        p.apiKey = key
    }
    return nil
}

func (p *CustomAuthPlugin) Wrap(handler fiber.Handler) fiber.Handler {
    return func(c *fiber.Ctx) error {
        key := c.Get("X-API-Key")
        if key != p.apiKey {
            return c.Status(401).JSON(fiber.Map{"error": "Invalid API key"})
        }
        return handler(c)
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

    // Import your custom plugins
    customplugins "yourapp/plugins"
)

func init() {
    // Register built-in plugins
    pluginloader.RegisterRoutePluginFactory("auth", authplugin.NewPlugin)
    pluginloader.RegisterGlobalPluginFactory("logger", loggerplugin.NewPlugin)

    // Register custom plugins
    pluginloader.RegisterGlobalPluginFactory("myplugin", customplugins.NewMyPlugin)
}

func main() {
    cfg := gorest.Config{
        ConfigPath:     ".",
        RegisterRoutes: resources.RegisterGeneratedRoutes,
    }
    gorest.Start(cfg)
}
```

Plugins are configured in `gorest.yaml` and loaded automatically by the core library.

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
├── formatter/              # JSON-LD formatting
├── generator/              # Code generation
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
└── cmd/                    # CLI generators
    ├── modelgen/          # Model generation
    ├── resourcegen/       # Resource generation
    └── openapigen/        # OpenAPI generation
```

---

## 🛠 Development Commands

```bash
# Generation
make modelgen         # Generate models
make resourcegen      # Generate resources & DTOs
make openapigen       # Generate OpenAPI
make generate         # Run all generators

# Testing
make test-up          # Start test database
make test-schema      # Load schema
make test             # Run tests
make build            # Build binary
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

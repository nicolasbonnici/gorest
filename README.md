# GoREST

[![Test](https://github.com/nicolasbonnici/gorest/actions/workflows/test.yml/badge.svg?branch=trunk)](https://github.com/nicolasbonnici/gorest/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/nicolasbonnici/gorest.svg)](https://pkg.go.dev/github.com/nicolasbonnici/gorest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/nicolasbonnici/gorest)](https://github.com/nicolasbonnici/gorest/blob/HEAD/go.mod)
[![Release](https://img.shields.io/github/v/release/nicolasbonnici/gorest)](https://github.com/nicolasbonnici/gorest/releases)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

🚀 **GoREST** is a Go library for building type-safe REST APIs in Go from your existing database schema or from scratch.

> [!WARNING]
> **Pre-1.0 Development**: The API is not yet stabilized and may introduce breaking changes until v1.0.0 is released. Pin to a specific version tag and test thoroughly before production use.

---

## ✨ Features

- ⚡ Type-safe generic CRUD operations
- 🚀 Unified processor module eliminating handler boilerplate: handlers become one-liners with automatic RBAC, validation, pagination, filtering & serialization
- 🔧 Fluent SQL query builder with database abstraction
- 🔑 Built-in JWT authentication with registration, login, and rotating refresh tokens
- 🔐 Role based access control with field-level permissions and a voter system
- 🧐 [Audit log](RBAC.md) of role assignments
- ✅ Security best practices, rate limiting, CORS, response compression and many more configurable core middleware
- 🎭 Hook layer to add your business logic and override any API layer
- 🧩 Modular plugin system that can add features, override some or all existing endpoints or even CLI commands
- 🛠 Code generation plugin for REST endpoints, DTOs and models ([gorest-codegen](https://github.com/nicolasbonnici/gorest-codegen))
- 🌐 JSON-LD support with semantic web context (@context, @type, @id)
- 🔗 Advanced resource deserialization with IRI and optional on demand relations
- 🔍 Advanced serialization, filtering & ordering
- 📄 Page based pagination with Hydra collections
- 👨🏻‍💻 DAL, migrations and fixtures with PostgreSQL, MySQL and SQLite engines support
- 🛡️ Production grade errors and processes management
- 🏷️ API versioning
- 📜 OpenAPI 3 spec generation ([gorest-openapi](https://github.com/nicolasbonnici/gorest-openapi))
- 💚 Status endpoint for health check ([gorest-status](https://github.com/nicolasbonnici/gorest-status))
- 🐳 Docker and Kubernetes support
- 🧪 Unit, integration and cross-database test suites

---

## 📚 Core Documentation

### Getting Started
- **[Configuration →](CONFIGURATION.md)** - YAML configuration, environment overrides, and templates
- **[Processor →](processor/README.md)** - Unified API processor eliminating CRUD boilerplate with one-liner handlers

### Data Management
- **[Query Builder →](query/README.md)** - Type-safe SQL query builder with fluent API
- **[Filtering & Ordering →](FILTERING.md)** - Advanced query filtering, comparison operators, and ordering
- **[Relation Expansion →](serializer/EXPAND_USAGE.md)** - Expand IRI references to full nested objects
- **[Serializer →](serializer/README.md)** - Flexible resource serializer with JSON-LD support
- **[Database Migrations →](migrations/README.md)** - Migration system with multi-database support
- **[Fixtures →](fixtures/README.md)** - Test fixture management

### CLI
- **[Migration CLI →](migrations/CLI.md)** - `migrate` CLI for running, rolling back, and repairing migrations
- **[User & Role CLI →](cmd/user/CLI.md)** - `user` CLI for managing users, assigning roles, and viewing role hierarchy

### Security & Access Control
- **[Authentication →](auth/README.md)** - Built-in JWT auth with rotating, database-backed refresh tokens
- **[Authorization →](AUTHORIZATION.md)** - Mandatory authorization layer and hook execution order
- **[RBAC →](RBAC.md)** - Role-based access control, voters, and field-level permissions
- **[Security Policy →](SECURITY.md)** - Reporting vulnerabilities and supported versions

### Business Logic
- **[DTOs & Field Control →](DTOS.md)** - Control resource attributes with `dto` tags
- **[Hooks System →](HOOKS.md)** - Lifecycle hooks for custom business logic
- **[Plugins →](PLUGINS.md)** - Plugin system, built-in plugins, and custom plugin creation

---

## 📦 GoREST usage

Import GoREST library directly in your Go projects:

```bash
go get github.com/nicolasbonnici/gorest@latest
```

### Available Packages

| Package | Description |
|---------|-------------|
| `processor` | Unified API processor eliminating handler boilerplate |
| `query` | Type-safe SQL query builder |
| `crud` | Type-safe CRUD operations with hooks |
| `database` | Multi-database abstraction |
| `filter` | Query filtering & ordering |
| `serializer` | JSON-LD serialization and relation expansion |
| `hooks` | Lifecycle hooks for business logic |
| `plugin` | Plugin interfaces |
| `pluginloader` | Plugin factory & loading |
| `pagination` | Hydra-compliant pagination |
| `response` | HTTP response helpers |
| `migrations` | Database migration system |
| `fixtures` | Test fixture management |

### Example

Use the packages on their own, without `gorest.Start`:

```go
package main

import (
    "log"

    "github.com/gofiber/fiber/v3"
    "github.com/nicolasbonnici/gorest/crud"
    "github.com/nicolasbonnici/gorest/database"
    _ "github.com/nicolasbonnici/gorest/database/postgres" // registers the driver
    "github.com/nicolasbonnici/gorest/query"
)

type User struct {
    ID     string `json:"id" db:"id"`
    Email  string `json:"email" db:"email"`
    Status string `json:"status" db:"status"`
}

func (User) TableName() string { return "users" }

func main() {
    db, err := database.Open("postgres", "postgres://...")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    users := crud.New[User](db)
    app := fiber.New()

    app.Get("/users", func(c fiber.Ctx) error {
        result, err := users.GetAllPaginated(c.Context(), crud.PaginationOptions{
            Limit:      10,
            Conditions: []query.Condition{query.Eq("status", "active")},
        })
        if err != nil {
            return fiber.ErrInternalServerError
        }
        return c.JSON(result.Items)
    })

    log.Fatal(app.Listen(":8000"))
}
```

📚 [Full API documentation on pkg.go.dev](https://pkg.go.dev/github.com/nicolasbonnici/gorest)

---

## 🔍 Quick Examples

### Query Builder

```go
import "github.com/nicolasbonnici/gorest/query"

// Build type-safe, parameterized SQL queries
sql, args, err := query.New(db.Dialect()).
    Select("id", "name", "email").
    From("users").
    Where(query.Eq("status", "active")).
    Where(query.Gt("age", 18)).
    OrderBy("created_at", query.DESC).
    Limit(10).
    Build()

// Use in hooks without string manipulation
func (h *PostHooks) ModifySelectQuery(ctx context.Context, op hooks.Operation, builder *query.SelectBuilder) (*query.SelectBuilder, bool) {
    if !isAuthenticated(ctx) {
        builder = builder.Where(query.Eq("status", "published"))
        return builder, true
    }
    return builder, false
}
```

📚 **[Full query builder documentation →](query/README.md)**

### Processor Pattern

```go
import "github.com/nicolasbonnici/gorest/processor"

// Create processor with configuration
proc := processor.New(processor.ProcessorConfig[
    models.Todo,
    dtos.TodoCreateDTO,
    dtos.TodoUpdateDTO,
    dtos.TodoResponseDTO,
]{
    DB:                 db,
    CRUD:               crud.New[models.Todo](db),
    Converter:          &converters.TodoConverter{},
    PaginationLimit:    20,
    PaginationMaxLimit: 100,
    AllowedFields:      []string{"id", "title", "status", "created_at"},
}).
    WithCreateHook(setTodoOwner) // func(c fiber.Ctx, dto dtos.TodoCreateDTO, todo *models.Todo) error

// Handlers become one-liners
type TodoResource struct {
    processor processor.Processor[models.Todo, dtos.TodoCreateDTO, dtos.TodoUpdateDTO, dtos.TodoResponseDTO]
}

func (r *TodoResource) Create(c fiber.Ctx) error  { return r.processor.Create(c) }
func (r *TodoResource) GetAll(c fiber.Ctx) error  { return r.processor.GetAll(c) }
func (r *TodoResource) GetByID(c fiber.Ctx) error { return r.processor.GetByID(c) }
func (r *TodoResource) Update(c fiber.Ctx) error  { return r.processor.Update(c) }
func (r *TodoResource) Delete(c fiber.Ctx) error  { return r.processor.Delete(c) }
```

📚 **[Full processor documentation →](processor/README.md)**

### Filtering & Ordering

```bash
# Filter by status
GET /todos?status=active

# Multiple filters with comparison
GET /todos?status=active&priority[gte]=5

# Order results
GET /todos?order[createdAt]=desc

# Combine all
GET /todos?status=active&priority[gte]=5&order[createdAt]=desc&limit=10
```

📚 **[Full filtering documentation →](FILTERING.md)**

### Expand Relations

```bash
# IRI reference (default)
GET /todos/123
# Returns: { "user": "/users/456", ... }

# Expand to full object
GET /todos/123?expand[]=user
# Returns: { "user": { "id": "456", "name": "Alice", ... }, ... }
```

📚 **[Full expansion documentation →](serializer/EXPAND_USAGE.md)**

### JSON-LD Support

```bash
# Regular JSON
curl -H "Accept: application/json" http://localhost:8000/todos/123

# JSON-LD with semantic context
curl -H "Accept: application/ld+json" http://localhost:8000/todos/123

# With API versioning enabled (api.versioning in gorest.yaml)
curl -H "Accept: application/ld+json" https://api.example.com/v2/todos/123
```

📚 **[Full JSON-LD documentation →](serializer/README.md)**

---

## 🚀 Quick Start

### 1. Create Your Project

GoREST requires Go 1.26+.

```bash
mkdir my-api && cd my-api
go mod init github.com/yourusername/my-api
go get github.com/nicolasbonnici/gorest@latest
go get github.com/nicolasbonnici/gorest-openapi@latest
```

### 2. Configure Your API

Create `gorest.yaml` in your project root:

```yaml
server:
  scheme: "${SERVER_SCHEME:-http}"
  host: "${SERVER_HOST:-localhost}"
  port: "${SERVER_PORT:-8000}"
  environment: "${ENV:-development}"
  cors_origins: "${CORS_ORIGINS:-*}"
  compression_enabled: true  # gzip/deflate/brotli support (default: true)
  compression_level: 2       # 1=speed, 2=balanced, 3=best compression
  body_limit: 4194304        # Max request body in bytes (default: 4 MiB). Raise it for large uploads.

database:
  url: "${DATABASE_URL}"

pagination:
  default_limit: "${PAGINATION_DEFAULT_LIMIT:-10}"
  max_limit: "${PAGINATION_MAX_LIMIT:-1000}"

auth:
  enabled: true
  jwt_secret: "${JWT_SECRET}"
  jwt_ttl: 900

plugins:
  - name: openapi
    enabled: true
    config:
      title: "My API"
      version: "1.0.0"
```

Set required environment variables:
```bash
export ENV="production"              # default: development
export DATABASE_URL="postgres://user:pass@localhost:5432/mydb?sslmode=require"
export SERVER_SCHEME="https"         # default: http
export SERVER_HOST="api.example.com" # default: localhost
export SERVER_PORT="8080"            # default: 8000
export JWT_SECRET=$(openssl rand -base64 32)
export CORS_ORIGINS="https://app.example.com" # default: *, origins need a scheme
export PAGINATION_DEFAULT_LIMIT="20" # default: 10
export PAGINATION_MAX_LIMIT="5000"   # default: 1000
```

Or use a `.env` file (dotenv support):
```bash
ENV=production
DATABASE_URL=postgres://user:pass@localhost:5432/mydb?sslmode=require
SERVER_SCHEME=https
SERVER_HOST=api.example.com
SERVER_PORT=8080
JWT_SECRET=your-secret-key-here
CORS_ORIGINS=https://app.example.com
PAGINATION_DEFAULT_LIMIT=20
PAGINATION_MAX_LIMIT=5000
```

📚 **[Full configuration documentation →](CONFIGURATION.md)**

#### Environment Variable Interpolation

GoREST supports bash-style environment variable interpolation with default fallback values:

**Syntax:**
- `${VAR}` - Use environment variable VAR (leaves `${VAR}` unchanged if not set)
- `${VAR:-default}` - Use environment variable VAR, or "default" if not set

**Examples:**
```yaml
server:
  # Will use environment variable or fallback to default
  port: "${SERVER_PORT:-8000}"
  host: "${SERVER_HOST:-localhost}"

  # Required variable (no default)
  environment: "${ENV}"

database:
  # Complex defaults work too
  url: "${DATABASE_URL:-postgres://user:pass@localhost:5432/dev?sslmode=disable}"

pagination:
  # Numeric defaults
  default_limit: "${PAGINATION_DEFAULT_LIMIT:-10}"
  max_limit: "${PAGINATION_MAX_LIMIT:-1000}"

auth:
  # Empty default
  jwt_secret: "${JWT_SECRET:-}"
```

**Behavior:**
- If the environment variable is set (even to an empty string), its value is used
- If the environment variable is not set and a default is provided, the default is used
- If the environment variable is not set and no default is provided, the original `${VAR}` string remains

**Note:** A quoted placeholder that resolves to a number or `true`/`false` is unquoted before parsing, so `"${SERVER_PORT:-8000}"` fills an integer field like `port` and `"${RATELIMIT_ENABLED:-true}"` a boolean one.

### 3. Create Your Main Application

You can either write your routes manually or use the **[gorest-codegen](https://github.com/nicolasbonnici/gorest-codegen)** plugin to generate them from your database schema.

**Manual approach:**
```go
package main

import (
    "github.com/gofiber/fiber/v3"
    "github.com/nicolasbonnici/gorest"
    "github.com/nicolasbonnici/gorest/crud"
    "github.com/nicolasbonnici/gorest/database"
    "github.com/nicolasbonnici/gorest/plugin"
    "github.com/nicolasbonnici/gorest/pluginloader"
    openapiplugin "github.com/nicolasbonnici/gorest-openapi"
)

// Plugins listed in gorest.yaml must be registered before gorest.Start.
func init() {
    pluginloader.RegisterPluginFactory("openapi", openapiplugin.NewPlugin)
}

type User struct {
    ID    string `json:"id" db:"id"`
    Email string `json:"email" db:"email"`
}

func (User) TableName() string { return "users" }

func main() {
    cfg := gorest.Config{
        ConfigPath: ".",
        RegisterRoutes: func(router fiber.Router, db database.Database, paginationLimit, paginationMaxLimit int, pluginRegistry *plugin.PluginRegistry) {
            userCRUD := crud.New[User](db)
            router.Get("/users", func(c fiber.Ctx) error {
                result, err := userCRUD.GetAllPaginated(c.Context(), crud.PaginationOptions{Limit: paginationLimit})
                if err != nil {
                    return fiber.ErrInternalServerError
                }
                return c.JSON(result.Items)
            })
        },
    }
    gorest.Start(cfg)
}
```

**With code generation plugin:**
```bash
# Install and run gorest-codegen
go run github.com/nicolasbonnici/gorest-codegen/cmd/codegen@latest all
```

```go
package main

import (
    "github.com/yourusername/my-api/generated/resources"
    "github.com/nicolasbonnici/gorest"
    "github.com/nicolasbonnici/gorest/pluginloader"
    openapiplugin "github.com/nicolasbonnici/gorest-openapi"
)

func init() {
    pluginloader.RegisterPluginFactory("openapi", openapiplugin.NewPlugin)
}

func main() {
    cfg := gorest.Config{
        ConfigPath:     ".",
        RegisterRoutes: resources.RegisterGeneratedRoutes,
    }
    gorest.Start(cfg)
}
```

### 4. Run Your API
```bash
go run main.go
```

Your API is now running at **http://localhost:8000/** (with the default `SERVER_*` values):
- 📚 API specs: **http://localhost:8000/openapi** (hidden when `environment` is `production`, unless the plugin sets `hide_on_production: false`)
- 💚 Status: **http://localhost:8000/status** (with [gorest-status](https://github.com/nicolasbonnici/gorest-status))
- 🔑 Auth: **http://localhost:8000/auth/register**, `/auth/login`, `/auth/refresh`, `/auth/logout`

Routes live at the root. See API Versioning below to serve them under a `/v1` style prefix.

---

## 📂 Project Structure

### Basic Project
```
my-api/
├── gorest.yaml              # Configuration
└── main.go                  # Your application
```

### With Code Generation Plugin (optional)
```
my-api/
├── gorest.yaml              # Configuration
├── main.go                  # Your application
└── generated/               # Generated by gorest-codegen plugin
    ├── models/              # DB models
    ├── resources/           # REST handlers
    ├── dtos/                # Data transfer objects
    └── openapi/             # OpenAPI schema
```

### GoREST Library
```
gorest/
├── processor/               # Unified API processor
├── auth/                    # Built-in authentication layer
├── rbac/                    # Role-based access control
├── crud/                    # Generic CRUD
├── query/                   # SQL query builder
├── database/                # Multi-DB abstraction
│   ├── postgres/
│   ├── mysql/
│   └── sqlite/
├── migrations/              # Migration system
├── fixtures/                # Fixture management
├── filter/                  # Query filtering
├── serializer/              # JSON-LD serialization, relation expansion
├── hooks/                   # Lifecycle hooks
├── plugin/                  # Plugin interfaces
├── pluginloader/            # Plugin loading
├── middleware/              # Core middleware
├── pagination/              # Hydra pagination
└── response/                # HTTP helpers
```

---

## 🚀 Production Deployment

### Docker

Health checks use the [gorest-status](https://github.com/nicolasbonnici/gorest-status) plugin.

```dockerfile
# Dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o api

FROM alpine:3
RUN apk --no-cache add ca-certificates curl
WORKDIR /app
COPY --from=builder /app/api .
COPY gorest.yaml .
CMD ["./api"]
```

```yaml
# docker-compose.yml
services:
  api:
    build: .
    ports: ["8000:8000"]
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - JWT_SECRET=${JWT_SECRET}
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/status"]
      interval: 30s
      timeout: 3s
      retries: 3
```

### Nginx Reverse Proxy

```nginx
upstream gorest {
    server localhost:8000;
}

server {
    listen 443 ssl;
    http2 on;
    server_name api.example.com;

    # SSL configuration
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    # Redirect root to the API docs
    location = / {
        return 301 https://$host/openapi;
    }

    location / {
        proxy_pass http://gorest;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Recommended headers. GoREST mints its own request id and logs this
        # one as client_request_id, so nginx and app log lines still join up.
        proxy_set_header X-Request-ID $request_id;
        proxy_buffering off;
    }
}
```

### Kubernetes Health Checks

```yaml
livenessProbe:
  httpGet:
    path: /status
    port: 8000
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /status
    port: 8000
  initialDelaySeconds: 5
  periodSeconds: 10
```

---

## 🏷️ API Versioning

Versioning is off by default and routes are served at the root (`/posts`, `/auth/login`, `/openapi`). Turn it on in `gorest.yaml` to prefix every route, including plugin and auth endpoints:

```yaml
api:
  versioning:
    enabled: true
    version: "v2"   # default: v1
```

```bash
curl http://localhost:8000/v2/posts
curl http://localhost:8000/v2/auth/login
```

To serve several API versions at once, run one instance per version and route by prefix:

```nginx
location /v1/ {
    proxy_pass http://localhost:8001;
}

location /v2/ {
    proxy_pass http://localhost:8002;
}
```

---

## 🔒 Security

- **Passwords**: bcrypt hashing with a length policy
- **JWT**: short-lived access tokens, 32+ character secrets required
- **Refresh tokens**: opaque, stored hashed, rotated on every use; replaying a revoked token revokes the whole family
- **Errors**: database and driver messages never reach clients
- **Request IDs**: always minted server-side; a client supplied `X-Request-ID` is only logged as `client_request_id`
- **CORS**: configurable origins
- **Rate Limiting**: configurable per-IP limits
- **SQL Injection**: parameterized queries through the query builder
- **Input Validation**: go-playground/validator support
- **Security Headers**: X-Frame-Options, CSP, HSTS, etc.

See [SECURITY.md](SECURITY.md) to report a vulnerability.

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

## 📋 Resources

- [Release history](https://github.com/nicolasbonnici/gorest/releases)
- [API reference on pkg.go.dev](https://pkg.go.dev/github.com/nicolasbonnici/gorest)

### Blog Posts

GoREST articles on [nbonnici.info](https://nbonnici.info/blog/?tag=gorest), most recent first:

- 2026-11-04: [GoREST: faster, stronger, safer](https://nbonnici.info/blog/gorest-faster-stronger-safer)
- 2025-12-04: [Dites bonjour à GoREST v0.3](https://nbonnici.info/blog/say-hello-to-gorest-v03-4k8k)
- 2025-11-15: [Comment créer une API REST en quelques secondes avec Go](https://nbonnici.info/blog/how-to-create-a-rest-api-in-seconds-59a4)
- 2025-11-12: [GoREST est mort...](https://nbonnici.info/blog/gorest-is-dead-1ap9)
- 2025-11-05: [GoREST turn any database to a production grade REST API](https://nbonnici.info/blog/from-a-database-to-a-rest-api-3cgf)

---

## 📜 License

MIT, free to use in your projects 🚀

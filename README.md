# GoREST

[![Test](https://github.com/nicolasbonnici/gorest/actions/workflows/test.yml/badge.svg?branch=trunk)](https://github.com/nicolasbonnici/gorest/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nicolasbonnici/gorest)](https://goreportcard.com/report/github.com/nicolasbonnici/gorest)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

🚀 **GoREST** is a Go library for building type-safe REST APIs in Go from your existing database schema or from scratch.

## ✨ Features

- ⚡ Type-safe generic CRUD operations with hooks system
- 🔧 Fluent SQL query builder with database abstraction
- 🔐 Full DTO support with field-level control (`dto` tags)
- 🔑 JWT authentication with context-aware plugins ([gorest-auth](https://github.com/nicolasbonnici/gorest-auth))
- 🎭 Hook layer to add your business logic and override any API layer
- 🧩 Modular plugin system that can add features, override some or all existing endpoints or even CLI commands
- 🛠 **Code generation plugin** for REST endpoints, DTOs and models ([gorest-codegen](https://github.com/nicolasbonnici/gorest-codegen))
- ✅ Security best practices, rate limiting, CORS and many more configurable core middleware
- 🌐 JSON-LD support with semantic web context (@context, @type, @id)
- 🔗 Advanced resource deserialization with IRI and optional on demand relations
- 🔍 Advanced serialization, filtering & ordering
- 📄 Page based pagination with Hydra collections
- 👨🏻‍💻 DAL, migrations and fixtures with PostgreSQL, MySQL and SQLite engines support
- 🛡️ Production grade errors and processes management
- 🐳 Docker and Kubernetes support
- 🧪 Full test coverage with automated testing
- 💚 Status endpoint for health check ([gorest-status](https://github.com/nicolasbonnici/gorest-status))
- 📜 OpenAPI 3 spec generation ([gorest-openapi](https://github.com/nicolasbonnici/gorest-openapi))

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
server:
  scheme: "${SERVER_SCHEME:-http}"
  host: "${SERVER_HOST:-localhost}"
  port: "${SERVER_PORT:-8000}"
  environment: "${ENV:-development}"
  cors_origins: "${CORS_ORIGINS:-*}"


database:
  url: "${DATABASE_URL}"

pagination:
  default_limit: "${PAGINATION_DEFAULT_LIMIT:-10}"
  max_limit: "${PAGINATION_MAX_LIMIT:-1000}"

plugins:
  - name: auth
    enabled: true
    config:
      jwt_secret: "${JWT_SECRET}"
      jwt_ttl: 900
```

Set required environment variables:
```bash
export ENV="production"              # default: development
export DATABASE_URL="postgres://user:pass@localhost:5432/mydb?sslmode=require"
export SERVER_SCHEME="https"         # default: http
export SERVER_HOST="api.example.com" # default: localhost
export SERVER_PORT="8080"            # default: 8000
export JWT_SECRET=$(openssl rand -base64 32)
export CORS_ORIGINS="localhost:8000" # default: *
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
CORS_ORIGINS=example.com
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

plugins:
  - name: auth
    config:
      # Empty default
      jwt_secret: "${JWT_SECRET:-}"
```

**Behavior:**
- If the environment variable is set (even to an empty string), its value is used
- If the environment variable is not set and a default is provided, the default is used
- If the environment variable is not set and no default is provided, the original `${VAR}` string remains

**Note:** Environment variable interpolation only works for string fields in the configuration. Integer fields like `port`, `default_limit`, and `max_limit` must be specified as numeric values directly in the YAML file.

### 3. Create Your Main Application

You can either write your routes manually or use the **[gorest-codegen](https://github.com/nicolasbonnici/gorest-codegen)** plugin to generate them from your database schema.

**Manual approach:**
```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/nicolasbonnici/gorest"
    "github.com/nicolasbonnici/gorest/crud"
    "github.com/nicolasbonnici/gorest/database"
)

type User struct {
    ID    string `json:"id" db:"id"`
    Email string `json:"email" db:"email"`
}

func (User) TableName() string { return "users" }

func main() {
    cfg := gorest.Config{
        ConfigPath: ".",
        RegisterRoutes: func(app *fiber.App, db database.Database) {
            userCRUD := crud.New[User](db)
            app.Get("/users", func(c *fiber.Ctx) error {
                result, _ := userCRUD.GetAllPaginated(c.Context(), crud.PaginationOptions{Limit: 10})
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
)

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

Your API is now running at: **${SERVER_SCHEME}://${SERVER_HOST}:${SERVER_PORT}/**
- 📚 API specs: **${SERVER_SCHEME}://${SERVER_HOST}:${SERVER_PORT}/openapi**
- 💚 Status: **${SERVER_SCHEME}://${SERVER_HOST}:${SERVER_PORT}/status**

(With default values: **http://localhost:8000/**)

---

## 📦 GoREST usage

Import GoREST library directly in your Go projects:

```bash
go get github.com/nicolasbonnici/gorest@latest
```

### Available Packages

| Package | Description |
|---------|-------------|
| `query` | Type-safe SQL query builder |
| `crud` | Type-safe CRUD operations with hooks |
| `database` | Multi-database abstraction |
| `expand` | Relation expansion (IRI to object) |
| `filter` | Query filtering & ordering |
| `serializer` | JSON-LD response serialization |
| `hooks` | Lifecycle hooks for business logic |
| `plugin` | Plugin interfaces |
| `pluginloader` | Plugin factory & loading |
| `pagination` | Hydra-compliant pagination |
| `response` | HTTP response helpers |
| `migrations` | Database migration system |
| `fixtures` | Test fixture management |

### Example

```go
import (
    "github.com/gofiber/fiber/v2"
    "github.com/nicolasbonnici/gorest/database"
    "github.com/nicolasbonnici/gorest/crud"
    auth "github.com/nicolasbonnici/gorest-auth"
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

    app.Listen(":8000")
}
```

📚 [Full API documentation on pkg.go.dev](https://pkg.go.dev/github.com/nicolasbonnici/gorest)

---

## 📚 Core Documentation

### Configuration & Setup
- **[Configuration →](CONFIGURATION.md)** - YAML configuration, environment overrides, and templates

### Data Management
- **[Query Builder →](QUERY_BUILDER.md)** - Type-safe SQL query builder with fluent API
- **[Filtering & Ordering →](FILTERING.md)** - Advanced query filtering, comparison operators, and ordering
- **[Relation Expansion →](serializer/EXPAND_USAGE.md)** - Expand IRI references to full nested objects
- **[Serializer →](serializer/README.md)** - Flexible resource serializer with JSON-LD support
- **[Database Migrations →](migrations/README.md)** - Migration system with multi-database support
- **[Fixtures →](fixtures/README.md)** - Test fixture management

### Business Logic
- **[DTOs & Field Control →](DTOS.md)** - Control resource attributes with `dto` tags
- **[Hooks System →](HOOKS.md)** - Lifecycle hooks for custom business logic
- **[Plugins →](PLUGINS.md)** - Plugin system, built-in plugins, and custom plugin creation

---

## 🔍 Quick Examples

### Query Builder

```go
import "github.com/nicolasbonnici/gorest/query"

// Build type-safe SQL queries
sql, args := query.New(db.Dialect()).
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

📚 **[Full query builder documentation →](QUERY_BUILDER.md)**

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
```

📚 **[Full JSON-LD documentation →](serializer/README.md)**

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
├── crud/                    # Generic CRUD
├── database/                # Multi-DB abstraction
│   ├── postgres/
│   ├── mysql/
│   └── sqlite/
├── migrations/              # Migration system
├── fixtures/                # Fixture management
├── expand/                  # Relation expansion
├── filter/                  # Query filtering
├── serializer/              # JSON-LD serialization
├── hooks/                   # Lifecycle hooks
├── plugin/                  # Plugin interfaces
├── pluginloader/            # Plugin loading
├── middleware/              # Core middleware
├── pagination/              # Hydra pagination
└── response/                # HTTP helpers
```

---

## 🛠 Development Commands

```bash
# Testing
make test-up          # Start test databases
make test-schema      # Load test schema
make test             # Run all tests
make test-coverage    # Run tests with coverage
```

**Code Generation Plugin:**
See [gorest-codegen](https://github.com/nicolasbonnici/gorest-codegen) for generating models, resources, DTOs and OpenAPI specs from your database schema.

---

## 🚀 Production Deployment

### Docker

Using GoREST status plugin [gorest-status](https://github.com/nicolasbonnici/gorest-status)

```yaml
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
```

### Nginx Reverse Proxy

```nginx
upstream gorest {
    server localhost:8000;
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
    path: /status
    port: 8000
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
- **Security Headers**: X-Frame-Options, CSP, HSTS, etc.

---

## Git Hooks

This directory contains git hooks for the GoREST project to maintain code quality.

### Available Hooks

#### pre-commit

Runs before each commit to ensure code quality:
- **Linting**: Runs `make lint` to check code style and potential issues
- **Tests**: Runs `make test` to verify all tests pass

### Installation

#### Automatic Installation

Run the install script from the project root:

```bash
./.githooks/install.sh
```

### Manual Installation

Copy the hooks to your `.git/hooks` directory:

```bash
cp .githooks/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

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

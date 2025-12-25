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
- 🔗 IRI relations with optional expansion (`expand[]=relation`)
- 🔍 Advanced filtering & ordering 
- 📄 Page based pagination with Hydra collections
- 👨🏻‍💻 DAL for PostgreSQL, MySQL and SQLite engines
- 🛡️ Production grade errors and processes management
- 🐳 Docker support with multi-database testing
- 🧪 Full test coverage with automated testing
- 💚 Status check endpoint (`/status`)
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
    
  enums:
    enabled: true

  auth:
    enabled: true
    # Default: all methods require authentication
    defaults:
      GET: true
      POST: true
      PUT: true
      DELETE: true
    # Per-resource overrides
    endpoints:
      - name: posts
        GET: false  # Public read - GET /posts and GET /posts/:id


server:
  port: 3000
  environment: "development"

database:
  url: "${DATABASE_URL}"

pagination:
  default_limit: 10
  max_limit: 1000

plugins:
  - name: ratelimit
    enabled: true
    config:
      requests_per_second: 100
      burst: 200
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
- 💚 Status: **http://localhost:3000/status**

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

  enums:
    enabled: true

  auth:
    enabled: true
    # Default: all methods require authentication
    defaults:
      GET: true
      POST: true
      PUT: true
      DELETE: true
    # Per-resource overrides
    endpoints:
      - name: posts
        GET: false  # Public read - GET /posts and GET /posts/:id
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
server:
  cors_origins: "*"

plugins:
  - name: status
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
  cors_origins: "https://app.example.com"

plugins:
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
| `expand` | Relation expansion (IRI to object) |
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

- **auth** - JWT authentication for protected routes

**External Plugins:**
- **status** - Status check endpoint with database connectivity monitoring - [gorest-status](https://github.com/nicolasbonnici/gorest-status)
- **openapi** - OpenAPI documentation UI and schema serving - [gorest-openapi](https://github.com/nicolasbonnici/gorest-openapi)
- **benchmark** - API performance benchmarking tool - [gorest-benchmark](https://github.com/nicolasbonnici/gorest-benchmark)

**Core Middleware:**
- **Security Headers** - Always enabled (X-Frame-Options, CSP, HSTS, etc. and TRACE method blocking)
- **CORS** - Always enabled (configure via `server.cors_origins` in YAML)
- **RequestID** - Always enabled (unique request ID tracking with UUID generation)
- **Logger** - Always enabled (HTTP request/response logging with structured logs)
- **ContentNegotiation** - Always enabled (validates Content-Type: application/json for POST/PUT/PATCH)
- **RateLimit** - Optional (per-IP rate limiting, configure via `server.ratelimit_*` in YAML)

### Configuration

Plugins are configured in `gorest.yaml` as a flat list:

```yaml
plugins:
  - name: requestid
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

    authplugin "github.com/nicolasbonnici/gorest/plugins/auth"

    customplugins "yourapp/plugins"
)

func init() {
    pluginloader.RegisterPluginFactory("auth", authplugin.NewPlugin)

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

See [PLUGINS.md](PLUGINS.md) for complete documentation including custom endpoint creation, CLI commands, and advanced patterns.

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

## 🔗 Expand Relations

Deserialize IRI references into full nested objects using the `expand[]` query parameter:

```bash
# Expand single relation
GET /todos?expand[]=user

# Expand multiple relations
GET /todos?expand[]=user&expand[]=comments

# Combine with filters and pagination
GET /todos?status=active&limit=10&expand[]=user
```

**Response without expand** (IRI reference):
```json
{
  "id": "todo-123",
  "user": "/users/user-456",
  "title": "Buy groceries"
}
```

**Response with expand** (full object):
```json
{
  "id": "todo-123",
  "user": {
    "id": "user-456",
    "name": "Alice",
    "email": "alice@example.com"
  },
  "title": "Buy groceries"
}
```

**Key features:**
- ✅ Clean relation names (`user`) instead of foreign keys (`userId`)
- ✅ Works with both JSON and JSON-LD formats
- ✅ Supports collections and single items
- ✅ Respects DTO field visibility rules

See [expand/USAGE.md](expand/USAGE.md) and [expand/EXAMPLES.md](expand/EXAMPLES.md) for complete documentation.

---

## 🗄️ Database Migrations

GoREST includes a production-ready database migration system with multi-database support, plugin migrations, and comprehensive safety features.

### Features

- ✅ **Multi-Database Support**: PostgreSQL, MySQL, SQLite
- ✅ **Dialect-Specific Migrations**: Database-specific SQL files with generic fallback
- ✅ **Plugin System Integration**: Each plugin can maintain its own migrations
- ✅ **Checksum Verification**: Prevents migration drift between environments
- ✅ **Advisory Locking**: Prevents concurrent execution
- ✅ **Transaction Safety**: Automatic rollback on failure
- ✅ **Dependency Resolution**: Plugin migrations run in correct order

### Quick Start

**1. Create Migration Files**

Migration files use timestamp-based naming: `{timestamp}_{name}.{up|down}[.{dialect}].sql`

```bash
# Generate timestamp
date +%Y%m%d%H%M%S
# Output: 20250120143022
```

**2. Write Migrations**

Create dialect-specific migration files:

```
migrations/
├── 20250120143022_create_users.up.postgres.sql
├── 20250120143022_create_users.down.postgres.sql
├── 20250120143022_create_users.up.mysql.sql
├── 20250120143022_create_users.down.mysql.sql
├── 20250120143022_create_users.up.sqlite.sql
└── 20250120143022_create_users.down.sqlite.sql
```

**Example (PostgreSQL):**

```sql
-- 20250120143022_create_users.up.postgres.sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_user_email ON users (email);
```

```sql
-- 20250120143022_create_users.down.postgres.sql
DROP INDEX IF EXISTS idx_user_email;
DROP TABLE IF EXISTS users CASCADE;
```

**3. Embed and Run Migrations**

```go
package main

import (
    "context"
    "embed"
    "github.com/nicolasbonnici/gorest/database"
    "github.com/nicolasbonnici/gorest/migrations"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func main() {
    db, _ := database.Open("postgres", "postgres://localhost/mydb")
    defer db.Close()

    // Create migration source
    appSource := migrations.NewEmbeddedSource("app", migrationFiles, "migrations", db)

    // Create migrator
    migrator := migrations.NewMigrator(db, appSource)

    // Run all pending migrations
    if err := migrator.Up(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

### Migration Commands

```go
ctx := context.Background()

// Apply all pending migrations
migrator.Up(ctx)

// Apply next pending migration
migrator.UpOne(ctx)

// Revert last migration
migrator.Down(ctx)

// Check migration status
statuses, _ := migrator.Status(ctx)
```

### Plugin Migrations

Plugins can provide their own migrations:

```go
package auth

import (
    "embed"
    "github.com/nicolasbonnici/gorest/migrations"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func (p *AuthPlugin) MigrationSource() plugin.MigrationSource {
    return migrations.NewEmbeddedSource("auth", migrationFiles, "migrations", p.db)
}

func (p *AuthPlugin) MigrationDependencies() []string {
    return []string{"app"} // Auth depends on app migrations
}
```

See [migrations/README.md](migrations/README.md) for complete documentation including:
- Detailed API reference
- Safety features (checksums, locking, dirty database detection)
- Plugin migration examples
- Error handling and recovery
- Best practices and troubleshooting

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
  "user": "/users/def-456",
  "title": "Buy groceries"
}
```

Foreign keys automatically convert to clean relation names with IRI values (`userId` → `user: "/users/def-456"`).

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
├── migrations/             # Database migration system
├── expand/                 # Relation expansion
├── filter/                 # Query filtering
├── serializer/             # JSON-LD serialization
├── codegen/                # Code generation
├── health/                 # Health check endpoint
├── hooks/                  # Lifecycle hooks
├── logger/                 # Logging utilities
├── pagination/             # Hydra pagination
├── plugin/                 # Plugin interfaces (core)
├── pluginloader/           # Plugin factory & loading system
├── plugins/                # Built-in plugin implementations
│   ├── auth/              # JWT authentication (with migrations example)
│   ├── contenttype/       # Content-Type validation
│   └── ratelimit/         # Rate limiting
├── middleware/             # Core middleware (security, CORS, requestid, logger, etc.)
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
      test: ["CMD", "curl", "-f", "http://localhost:3000/status"]
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
    path: /status
    port: 3000
  initialDelaySeconds: 10
  periodSeconds: 30
```

---

## 🔒 Security

- **Passwords**: bcrypt hashing with automatic salts
- **JWT**: 32+ character secrets required
- **CORS**: Configurable origins (core middleware, always enabled)
- **Rate Limiting**: Configurable per-IP limits
- **SQL Injection**: Parameterized queries
- **Input Validation**: go-playground/validator support
- **Security Headers**: X-Frame-Options, CSP, HSTS, etc. (core middleware, always enabled)

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

# gorest

🚀 **gorest** is a code generator for PostgreSQL REST APIs in Go.
It introspects your database schema and generates type-safe **CRUD endpoints automatically**.

## ✨ Features
- 🔎 Auto-discovery of tables, columns & types
- 🛠 Generated CRUD endpoints for each table
- 🔐 Full DTO support with automatic sensitive field exclusion
- 🔑 JWT authentication with decorator pattern
- 📜 OpenAPI 3.0 spec generation
- 🐳 Docker support
- ⚡ Type-safe generic CRUD operations
- 🧪 Full test coverage with automated testing
- 💚 Health check endpoint (`/health`)
- 🛡️ Graceful shutdown handling

---

## ⚙️ Requirements
- Go **1.25+**
- Docker & Docker Compose
- PostgreSQL **18+**

---

## 🚀 Quick Start

### 1. Clone & Setup
```bash
git clone https://github.com/nicolasbonnici/gorest.git
cd gorest
```

### 2. Configure Environment
```bash
cp .env.example .env
# Edit .env with your database connection details
```

Required environment variables:
- `DATABASE_URL` - PostgreSQL connection string
- `JWT_SECRET` - Secret key for JWT authentication
- `PORT` - Server port (default: 3000)

### 3. Start Test Database
```bash
make test-up
make test-schema
```

### 3. Generate Code
```bash
# Generate all code (models → resources → openapi)
make generate

# Or run individually:
make modelgen      # Generate models from database schema
make resourcegen   # Generate REST API resources
make openapigen    # Generate OpenAPI schema
```

This generates:
- `internal/api/models/*.go` - Type-safe model structs
- `internal/api/dtos/*.go` - Data Transfer Objects (Create/Update/Response)
- `internal/api/resources/*.go` - REST API endpoints with DTO conversion
- `internal/api/routes.go` - Auto-generated route registration
- `internal/api/openapi/*.go` - OpenAPI schema stubs

### 4. Build & Run
```bash
make build
./bin/gorest
```

API available at: **http://localhost:3000**
Health check: **http://localhost:3000/health**
OpenAPI API specs: **http://localhost:3000/openapi.json**

---

## 📂 Project Structure
```
gorest/
├── cmd/                      # CLI tools
│   ├── modelgen/main.go      # Model generator CLI
│   ├── resourcegen/main.go   # Resource generator CLI
│   └── openapigen/main.go    # OpenAPI generator CLI
├── pkg/
│   └── gorest/main.go        # API server entrypoint
├── internal/                 # Core logic
│   ├── modelgen.go           # Model generation logic
│   ├── apigen.go             # REST API generation logic
│   ├── auth.go               # JWT authentication
│   ├── openapi.go            # OpenAPI spec setup
│   ├── utils.go              # Shared utilities
│   ├── crud/                 # Generic CRUD operations
│   │   ├── crud.go           # Type-safe CRUD implementation
│   │   └── model.go          # Model interface
│   ├── hooks/                # Business logic hooks
│   │   ├── hooks.go          # Hook interfaces
│   │   └── factory.go        # Hook registry
│   ├── middleware/           # HTTP middleware
│   │   └── logger.go         # Request/response logging
│   ├── formatter/            # Response formatters
│   │   └── formatter.go      # JSON/JSON-LD formatters
│   └── api/                  # Generated code (gitignored)
│       ├── models/           # Database models
│       ├── dtos/             # Data Transfer Objects
│       ├── resources/        # REST endpoints
│       ├── openapi/          # OpenAPI schema stubs
│       └── routes.go         # Route registration
├── test/
│   └── sql/schema.sql        # Test database schema
├── config/
│   └── auth.json             # Authentication configuration
├── Makefile
├── compose.yml
├── HOOKS.md                  # Hooks documentation
└── .github/workflows/        # CI/CD
```

---

## 🛠 Development Commands

```bash
# Code Generation
make modelgen         # Generate models from database schema
make resourcegen      # Generate REST API resources
make openapigen       # Generate OpenAPI schema
make generate         # Run all generators in order

# Testing & Build
make test-up          # Start test database
make test-schema      # Load database schema
make build            # Build binary
make test             # Run all tests
```

---

## 📚 How It Works

1. **Schema Introspection**: Reads PostgreSQL `information_schema` to discover tables and columns
2. **Model Generation** (`make modelgen`): Creates Go structs with proper types & JSON tags
3. **Resource Generation** (`make resourcegen`): Creates REST endpoints with Fiber handlers using generic CRUD
   - Validates that models exist before generation
4. **OpenAPI Generation** (`make openapigen`): Creates OpenAPI schema stubs
5. **Server Startup**: Validates all generated files exist before starting the API server

### Code Generation Architecture

The generators are separate CLI tools that enforce proper ordering:
- **cmd/modelgen** → generates `internal/api/models/`
- **cmd/resourcegen** → generates `internal/api/resources/` (requires models)
- **cmd/openapigen** → generates `internal/api/openapi/`

This separation allows:
- Running generators independently during development
- Clear dependency management (resources depend on models)
- Better testing and validation of each generation step

---

## 🪝 Hooks System

gorest provides a powerful hooks system to customize business logic without modifying generated code. Hooks allow you to:

- **Validate and transform data** before/after database operations
- **Override SQL queries** for custom filtering or joins
- **Add authentication/authorization logic** per resource
- **Serialize responses** before sending to clients

### Hook Types

1. **StateProcessor** - Validate/enrich models before Create/Update/Delete
2. **SQLQueryListener** - Intercept queries before/after execution
3. **SQLQueryOverride** - Replace default queries with custom SQL
4. **Serializer** - Transform data before API response

### Quick Example

```go
type TodoHooks struct {
    hooks.NoOpHooks[models.Todo]
}

func (h *TodoHooks) StateProcessor(ctx context.Context, operation hooks.Operation, id any, todo *models.Todo) error {
    if operation == hooks.OperationCreate {
        // Validate title length
        if len(todo.Title) < 3 {
            return fmt.Errorf("title must be at least 3 characters")
        }
        // Enrich with user ID from context
        if userID := ctx.Value("user_id"); userID != nil {
            todo.UserID = userID.(string)
        }
    }
    return nil
}
```

For complete documentation, see [HOOKS.md](HOOKS.md)

---

## 🔐 DTOs & Security

gorest automatically generates Data Transfer Objects (DTOs) for enhanced security and API clarity:

### DTO Types

1. **CreateDTO** - For POST requests
   - Excludes auto-generated fields (id, created_at, updated_at)
   - Used when creating new resources

2. **UpdateDTO** - For PUT requests
   - Excludes auto-generated fields (id, created_at, updated_at)
   - Used when updating existing resources

3. **ResponseDTO** - For all responses
   - Automatically excludes sensitive fields
   - Protects passwords, tokens, and API keys

### Automatic Sensitive Field Exclusion

Sensitive fields are automatically detected and excluded from responses:
- `password`, `hashed_password`, `password_hash`
- `token`, `refresh_token`, `access_token`, `api_key`
- `secret`, and any field containing these keywords

### Example

**Model** (internal/api/models/user.go):
```go
type User struct {
    Id        string     `json:"id" db:"id"`
    Email     string     `json:"email" db:"email"`
    Password  *string    `json:"password" db:"password"`
    CreatedAt *time.Time `json:"created_at" db:"created_at"`
}
```

**Generated DTOs** (internal/api/dtos/user.go):
```go
// For creating users (POST /users)
type UserCreateDTO struct {
    Email    string  `json:"email"`
    Password *string `json:"password"`
    // id, created_at excluded
}

// For responses (GET /users)
type UserDTO struct {
    Id        string     `json:"id"`
    Email     string     `json:"email"`
    // Password automatically excluded (security)
    CreatedAt *time.Time `json:"created_at"`
}
```

**Conversion** (automatic in generated resources):
```go
// POST /users - accepts UserCreateDTO
func (r *UserResource) Create(c *fiber.Ctx) error {
    var createDTO dtos.UserCreateDTO
    c.BodyParser(&createDTO)

    // Convert to model
    user := createDTOToModel(createDTO)
    r.CRUD.Create(c.Context(), user)

    // Convert to response DTO (password excluded)
    dto := modelToUserDTO(user)
    return c.JSON(dto)
}
```

---

## 💚 Health Check

The `/health` endpoint provides real-time health status:

```bash
curl http://localhost:3000/health
```

**Healthy Response** (200 OK):
```json
{
  "status": "healthy",
  "database": {
    "status": "up"
  }
}
```

**Unhealthy Response** (503 Service Unavailable):
```json
{
  "status": "unhealthy",
  "database": {
    "status": "down",
    "error": "connection refused"
  }
}
```

---

## 🛡️ Graceful Shutdown

gorest handles shutdown signals gracefully:
- Listens for `SIGTERM` and `SIGINT` (Ctrl+C)
- 30-second timeout for in-flight requests
- Cleanly closes database connections
- Prevents data corruption during shutdown

---

## 📋 Changelog

See [CHANGELOG.md](CHANGELOG.md) for release history and migration guides.

---

## 📜 License
MIT – free to use in your projects 🚀
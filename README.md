# gorest

🚀 **gorest** is a code generator for PostgreSQL REST APIs in Go.
It introspects your database schema and generates type-safe **CRUD endpoints automatically**.

## ✨ Features
- 🔎 Auto-discovery of tables, columns & types
- 🛠 Generated CRUD endpoints for each table
- 🔑 JWT authentication
- 📜 OpenAPI 3.0 spec generation
- 🐳 Docker support
- ⚡ Type-safe generic CRUD operations
- 🧪 Full test coverage with automated testing

---

## ⚙️ Requirements
- Go **1.23+**
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
- `internal/api/resources/*.go` - REST API endpoints
- `internal/api/openapi/*.go` - OpenAPI schema stubs

### 4. Build & Run
```bash
make build
./bin/gorest
```

API available at: **http://localhost:3000**
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
│   │   ├── factory.go        # Hook registry
│   │   ├── todo.go           # Todo hooks (local)
│   │   └── user.go           # User hooks (local)
│   ├── middleware/           # HTTP middleware
│   │   └── logger.go         # Request/response logging
│   ├── formatter/            # Response formatters
│   │   └── formatter.go      # JSON/JSON-LD formatters
│   └── api/                  # Generated code (gitignored)
│       ├── models/           # Database models
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

## 📜 License
MIT – free to use in your projects 🚀
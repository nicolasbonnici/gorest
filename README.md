# gorest

🚀 **gorest** is a code generator for PostgreSQL REST APIs in Go.
It introspects your database schema and generates type-safe **CRUD endpoints automatically**.

## ✨ Features
- 🔎 Auto-discovery of tables, relations, columns & types
- 🛠 Generated CRUD endpoints for each table
- 🔐 Full DTO support with customizable serialization
- 🔑 JWT authentication with decorator pattern
- ⚡ Type-safe generic CRUD operations
- 🐳 Docker support
- 🧪 Full test coverage with automated testing
- 🛡️ Graceful shutdown handling
- 💚 Health check endpoint (`/health`)
- 📜 OpenAPI 3.0 spec generation

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

## 🔐 DTOs & Field Control

gorest automatically generates Data Transfer Objects (DTOs) for enhanced security and API clarity:

### DTO Types

1. **CreateDTO** - For POST requests
   - Excludes auto-generated fields (id, created_at, updated_at)
   - Used when creating new resources

2. **UpdateDTO** - For PUT requests
   - Excludes auto-generated fields (id, created_at, updated_at)
   - Used when updating existing resources

3. **ResponseDTO** - For all responses
   - Can be configured to exclude specific fields using DTO tags

### Controlling Field Visibility

Use the `dto` struct tag to control which fields appear in different contexts:

- `dto:"-"` - Exclude field completely from all DTOs
- `dto:"read"` - Only in response DTOs (GET requests)
- `dto:"write"` - Only in create/update DTOs (POST/PUT requests)
- `dto:"read,write"` or no tag - Include in all DTOs (default)

### Example

**Model** (internal/api/models/user.go):
```go
type User struct {
    Id        string     `json:"id" db:"id"`
    Email     string     `json:"email" db:"email"`
    Password  *string    `json:"password" db:"password" dto:"write"`
    ApiKey    *string    `json:"api_key" db:"api_key" dto:"-"`
    CreatedAt *time.Time `json:"created_at" db:"created_at"`
}
```

**Generated DTOs** (internal/api/dtos/user.go):
```go
// For creating users (POST /users)
type UserCreateDTO struct {
    Email    string  `json:"email"`
    Password *string `json:"password"`  // Included (dto:"write")
    // id, created_at, api_key excluded
}

// For responses (GET /users)
type UserDTO struct {
    Id        string     `json:"id"`
    Email     string     `json:"email"`
    CreatedAt *time.Time `json:"created_at"`
    // Password excluded (dto:"write" - write-only)
    // ApiKey excluded (dto:"-" - completely hidden)
}
```

**Conversion** (automatic in generated resources):
```go
// POST /users - accepts UserCreateDTO
func (r *UserResource) Create(c *fiber.Ctx) error {
    var createDTO dtos.UserCreateDTO
    c.BodyParser(&createDTO)

    // Convert to model
    user := userCreateDTOToModel(createDTO)
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

## 🤝 Contributing

GoRESTWe welcome contributions from developers of all experience levels! Whether you're fixing bugs, adding features, improving documentation, or sharing ideas, your input helps make **gorest** better for everyone.

### 🌟 Why Contribute?

- **Learn & Grow**: Get hands-on experience with Go, SQL, code generation, and REST API design
- **Real Impact**: Your code will help teams build APIs faster and more reliably
- **Community**: Join a growing community of developers passionate about developer tooling
- **Free Software**: Contribute to the free software philosophy

### 🚀 Quick Contribution Guide

#### 1. **Fork & Clone**
```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR_USERNAME/gorest.git
cd gorest
git checkout -b feature/your-awesome-feature
```

#### 2. **Set Up Development Environment**
```bash
# Install dependencies and start test database
make test-up
make test-schema

# Run tests to ensure everything works
make test
```

#### 3. **Make Your Changes**
- Write clean, well-documented code
- Follow existing code style and patterns
- Add tests for new features or bug fixes
- Update documentation if needed

#### 4. **Test Your Changes**
```bash
# Run all tests
make test

# Test code generation
make generate

# Test the API server
make build
./bin/gorest
```

#### 5. **Submit a Pull Request**
```bash
git add .
git commit -m "feat: your awesome feature description"
git push origin feature/your-awesome-feature
```

Then open a PR on GitHub with:
- Clear description of what you changed and why
- Screenshots/examples if applicable
- Reference to any related issues

### 💡 Contribution Ideas

Not sure where to start? Here are some ideas:

#### 🐛 **Bug Fixes**
- Fix edge cases in code generation
- Improve error handling and messages
- Resolve issues from the [issue tracker](https://github.com/nicolasbonnici/gorest/issues)

#### ✨ **New Features**
- Add support for more SQL types (arrays, JSONB, enums)
- Implement filtering, sorting, and pagination
- Create relationship/join support for nested resources
- Add support for other databases (MySQL, SQLite)
- Implement rate limiting middleware

#### 📚 **Documentation**
- Write tutorials and examples
- Create video guides or blog posts
- Improve inline code comments
- Add architecture diagrams
- Translate documentation

#### 🧪 **Testing**
- Increase test coverage
- Add integration tests
- Create benchmarks
- Test with different PostgreSQL versions

#### 🎨 **Developer Experience**
- Improve CLI output and formatting
- Add progress indicators for generators
- Create interactive setup wizard
- Build web UI for managing generated APIs

### 📝 Contribution Guidelines

- **Code Style**: Follow standard Go conventions (use `gofmt`, `golint`)
- **Commit Messages**: Use [conventional commits](https://www.conventionalcommits.org/) format
  - `feat:` for new features
  - `fix:` for bug fixes
  - `docs:` for documentation
  - `test:` for tests
  - `refactor:` for code improvements
  - `chore:` for chore tasks
- **Testing**: All new code should include tests
- **Documentation**: Update README/docs for user-facing changes
- **Breaking Changes**: Clearly document any breaking changes in your PR

### 🤔 Questions or Ideas?

- 💬 **Open an Issue**: Share your ideas or ask questions
- 🐛 **Report Bugs**: Help us improve by reporting issues you encounter
- 💡 **Propose Features**: Suggest new features via GitHub Discussions
- 📧 **Contact**: Reach out to maintainers for major contributions

### 🏆 Contributors

Thanks to all our amazing contributors! Your contributions make this project possible.

<!-- Contributors list will be auto-generated -->

**Ready to contribute?** [Fork the repo](https://github.com/nicolasbonnici/gorest/fork) and make your first PR today! 🎉

---

## 📋 Changelog

See [CHANGELOG.md](CHANGELOG.md) for release history and migration guides.

---

## 📜 License
MIT – free to use in your projects 🚀
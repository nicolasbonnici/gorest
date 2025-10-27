# gorest

🚀 **gorest** is a code generator for PostgreSQL REST APIs in Go.
It introspects your database schema and generates type-safe **CRUD endpoints automatically**.

## ✨ Features
- 🔎 Auto-discovery of tables, relations, columns & types
- ⚡ Type-safe generic CRUD operations with hooks system
- 🛠 Scaffold REST endpoints for each table
- 🔐 Full DTO support with field-level control (`dto` tags)
- 🔑 JWT authentication with context-aware middleware
- 🎭 Auto-population of fields from authentication context
- 🌐 JSON-LD support with semantic web context (@context, @type, @id)
- 🔗 Automatic foreign key to IRI conversion (e.g., `/users/{uuid}`)
- 👨🏻‍💻 DAL for PostgreSQL, MySQL and SQLite engines
- 🛡️ Production grade errors and processes management
- 🐳 Docker support with multi-database testing
- 🧪 Full test coverage with automated testing
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
- `internal/models/*.go` - Type-safe model structs
- `internal/api/dtos/*.go` - Data Transfer Objects (Create/Update/Response)
- `internal/api/resources/*.go` - REST API endpoints with DTO conversion
- `internal/api/routes.go` - Auto-generated route registration
- `internal/openapi/*.go` - OpenAPI schema stubs

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
│   ├── gorest/main.go        # API server entrypoint
│   └── database/             # Database abstraction layer
│       ├── database.go       # Core interface
│       ├── postgres/         # PostgreSQL implementation
│       ├── mysql/            # MySQL implementation
│       └── sqlite/           # SQLite implementation
├── internal/                 # Core logic
│   ├── modelgen.go           # Model generation logic
│   ├── apigen.go             # REST API generation logic
│   ├── auth.go               # JWT authentication
│   ├── openapi.go            # OpenAPI spec setup
│   ├── crud/                 # Generic CRUD operations
│   │   ├── crud.go           # Type-safe CRUD with hooks
│   │   └── model.go          # Model interface
│   ├── hooks/                # Business logic hooks
│   │   ├── hooks.go          # Hook interfaces
│   │   ├── factory.go        # Hook registry
│   │   ├── user.go           # User resource hooks
│   │   └── todo.go           # Todo resource hooks
│   ├── helpers/              # HTTP helpers
│   │   ├── decorators.go     # Auth middleware & context
│   │   └── response.go       # Response utilities
│   ├── formatter/            # Response formatters
│   │   └── formatter.go      # JSON/JSON-LD with IRI conversion
│   ├── models/               # Generated database models
│   │   ├── user.go           # (tracked in git)
│   │   └── *.go              # (other models gitignored)
│   ├── api/                  # Generated API code
│   │   ├── dtos/             # Data Transfer Objects
│   │   ├── resources/        # REST endpoints
│   │   └── routes.go         # Route registration
│   └── openapi/              # Generated OpenAPI stubs
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
- **cmd/modelgen** → generates `internal/models/`
- **cmd/resourcegen** → generates `internal/api/resources/` and `internal/api/dtos/` (requires models)
- **cmd/openapigen** → generates `internal/openapi/`

This separation allows:
- Running generators independently during development
- Clear dependency management (resources depend on models)
- Better testing and validation of each generation step
- Regeneration without losing custom business logic in hooks

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

Here's the actual TodoHooks implementation from the project, showing auto-population of `user_id` from JWT authentication:

```go
// internal/hooks/todo.go
type TodoHooks struct {
    hooks.NoOpHooks[models.Todo]
}

func (h *TodoHooks) StateProcessor(ctx context.Context, operation hooks.Operation, id any, todo *models.Todo) error {
    switch operation {
    case hooks.OperationCreate:
        // Validate required fields
        if todo.Title == "" {
            return fmt.Errorf("title is required")
        }
        if len(todo.Title) < 3 {
            return fmt.Errorf("title must be at least 3 characters")
        }

        // Auto-populate user_id from authenticated user
        // This happens server-side from JWT token - client cannot spoof it
        if userID := ctx.Value("user_id"); userID != nil {
            if uid, ok := userID.(string); ok {
                todo.UserId = &uid
            }
        }

    case hooks.OperationUpdate:
        // Validate on updates too
        if todo.Title != "" && len(todo.Title) < 3 {
            return fmt.Errorf("title must be at least 3 characters")
        }
    }
    return nil
}
```

**Key Features**:
- ✅ Server-side field population from authentication context
- 🔒 Client cannot send or modify `user_id` (excluded from DTOs with `dto:"read"` tag)
- ✅ Validation runs before database operations
- ✅ Works seamlessly with JWT middleware

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

### Example: Todo with Auto-Populated UserId

This example shows how the Todo model uses `dto:"read"` to make `user_id` read-only, preventing clients from sending or modifying it:

**Model** (internal/models/todo.go):
```go
type Todo struct {
    Id        string     `json:"id,omitempty" db:"id"`
    UserId    *string    `json:"user_id,omitempty" db:"user_id" dto:"read"`  // 🔒 Read-only
    Title     string     `json:"title" db:"title"`
    Content   string     `json:"content" db:"content"`
    UpdatedAt *time.Time `json:"updated_at,omitempty" db:"updated_at"`
    CreatedAt *time.Time `json:"created_at,omitempty" db:"created_at"`
}
```

**Generated DTOs** (internal/api/dtos/todo.go):
```go
// For creating todos (POST /todos)
type TodoCreateDTO struct {
    Title   string `json:"title"`
    Content string `json:"content"`
    // user_id excluded - populated server-side from JWT
    // id, created_at, updated_at excluded - auto-generated
}

// For updating todos (PUT /todos/:id)
type TodoUpdateDTO struct {
    Title   string `json:"title"`
    Content string `json:"content"`
    // user_id excluded - cannot be changed
}

// For responses (GET /todos)
type TodoDTO struct {
    Id        string     `json:"id"`
    UserId    *string    `json:"user_id"`  // ✅ Included (dto:"read")
    Title     string     `json:"title"`
    Content   string     `json:"content"`
    UpdatedAt *time.Time `json:"updated_at"`
    CreatedAt *time.Time `json:"created_at"`
}
```

**How it Works** (automatic in generated resources + hooks):
```go
// POST /todos - client sends TodoCreateDTO (no user_id)
func (r *TodoResource) Create(c *fiber.Ctx) error {
    var createDTO dtos.TodoCreateDTO  // No user_id field
    c.BodyParser(&createDTO)

    // Convert to model
    todo := todoCreateDTOToModel(createDTO)

    // Hooks auto-populate user_id from JWT context
    r.CRUD.Create(helpers.ContextWithUser(c), todo)  // 🔒 Server-side population

    // Response includes user_id
    dto := modelToTodoDTO(todo)  // TodoDTO has user_id
    return c.JSON(dto)
}
```

**Security Benefits**:
- 🔒 Client cannot send `user_id` in POST/PUT requests
- ✅ Server populates `user_id` from authenticated JWT token
- 🎯 Prevents users from creating/modifying resources for other users
- ✅ API responses correctly show the `user_id` value

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

## 🌐 JSON-LD Support

gorest automatically supports **JSON-LD** (Linked Data) format, providing semantic web context to your API responses. This makes your API machine-readable and interoperable with semantic web technologies.

### What is JSON-LD?

JSON-LD adds semantic context to regular JSON, making data self-describing and linked:

**Regular JSON** (application/json):
```json
{
  "id": "bc46c7ef-6191-4285-ae3a-3c90840bacee",
  "user_id": "a134d103-910d-4b81-9bff-293ba8d103d9",
  "title": "Buy groceries",
  "content": "Milk, eggs, bread"
}
```

**JSON-LD** (application/ld+json):
```json
{
  "@context": "https://schema.org/",
  "@type": "TodoDTO",
  "@id": "/todos/bc46c7ef-6191-4285-ae3a-3c90840bacee",
  "id": "bc46c7ef-6191-4285-ae3a-3c90840bacee",
  "user_id": "/users/a134d103-910d-4b81-9bff-293ba8d103d9",
  "title": "Buy groceries",
  "content": "Milk, eggs, bread"
}
```

### Key Features

1. **Automatic Foreign Key IRIs**: Foreign keys like `user_id` are automatically converted to IRIs (Internationalized Resource Identifiers):
   - `"user_id": "a134d103..."` → `"user_id": "/users/a134d103..."`
   - Follows semantic web best practices
   - Enables hypermedia-driven APIs

2. **Semantic Context**: Every response includes:
   - `@context` - Links to vocabulary (Schema.org)
   - `@type` - Specifies the resource type (TodoDTO, UserDTO, etc.)
   - `@id` - Canonical IRI for the resource

3. **Content Negotiation**: Supports both formats:
   ```bash
   # Get regular JSON
   curl -H "Accept: application/json" http://localhost:3000/todos/123

   # Get JSON-LD
   curl -H "Accept: application/ld+json" http://localhost:3000/todos/123
   ```

### Implementation

The formatter automatically detects foreign keys and converts them to IRIs:

```go
// internal/formatter/formatter.go
func (f *Formatter) formatItem(item interface{}, baseType string) map[string]interface{} {
    itemMap := toMap(item)

    // Convert foreign keys to IRIs
    for key, value := range itemMap {
        if strings.HasSuffix(key, "_id") && key != "id" {
            if valueStr, ok := value.(string); ok && valueStr != "" {
                resourceName := pluralize(strings.TrimSuffix(key, "_id"))
                itemMap[key] = fmt.Sprintf("/%s/%s", resourceName, valueStr)
            }
        }
    }

    itemMap["@context"] = "https://schema.org/"
    itemMap["@type"] = baseType
    return itemMap
}
```

### Benefits

- ✅ **Discoverable APIs**: Clients can navigate relationships via IRIs
- ✅ **Semantic Clarity**: Types and contexts make data self-describing
- ✅ **Standards Compliance**: Compatible with semantic web tools
- ✅ **Zero Configuration**: Works automatically for all resources

---

## 🛡️ Graceful Shutdown

gorest handles shutdown signals gracefully:
- Listens for `SIGTERM` and `SIGINT` (Ctrl+C)
- 30-second timeout for in-flight requests
- Cleanly closes database connections
- Prevents data corruption during shutdown

---

## 🔐 Authentication & Context System

gorest provides a sophisticated context system that bridges JWT authentication with your business logic hooks, enabling secure server-side field population.

### How It Works

```
HTTP Request with JWT
        ↓
   RequireAuth Middleware
   (extracts claims)
        ↓
   c.Locals("authenticated_user")
   (stores in Fiber context)
        ↓
   ContextWithUser(c)
   (bridges to standard Go context)
        ↓
   ctx.Value("user_id")
   (available in hooks)
```

### Components

#### 1. JWT Authentication Middleware

The `RequireAuth` middleware validates JWT tokens and extracts claims:

```go
// internal/helpers/decorators.go
func RequireAuth(jwtSecret string, handler fiber.Handler) fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Validate JWT token
        token, err := jwt.Parse(tokenString, keyFunc)

        // Extract claims and store in context
        if claims, ok := token.Claims.(jwt.MapClaims); ok {
            user := &AuthenticatedUser{
                UserID:    claims["user_id"].(string),
                Email:     claims["email"].(string),
                Firstname: claims["firstname"].(string),
                Lastname:  claims["lastname"].(string),
            }
            c.Locals(UserContextKey, user)  // Store in Fiber context
        }

        return handler(c)
    }
}
```

#### 2. Context Bridge Helper

The `ContextWithUser` helper bridges Fiber context to standard Go context:

```go
// internal/helpers/decorators.go
func ContextWithUser(c *fiber.Ctx) context.Context {
    ctx := c.Context()  // Get standard Go context

    // Extract authenticated user from Fiber context
    if user := GetAuthenticatedUser(c); user != nil {
        // Add user_id to standard context (for hooks)
        return context.WithValue(ctx, "user_id", user.UserID)
    }

    return ctx
}
```

#### 3. Usage in Generated Resources

All generated resources use `ContextWithUser` when calling CRUD operations:

```go
// internal/api/resources/todo.go (generated)
func (r *TodoResource) Create(c *fiber.Ctx) error {
    var createDTO dtos.TodoCreateDTO
    c.BodyParser(&createDTO)

    item := todoCreateDTOToModel(createDTO)

    // Pass context with user info to CRUD/hooks
    ctx := helpers.ContextWithUser(c)
    r.CRUD.Create(ctx, item)  // ← Context includes user_id

    // ...
}
```

#### 4. Hooks Access the Context

Hooks can then access `user_id` from the context:

```go
// internal/hooks/todo.go
func (h *TodoHooks) StateProcessor(ctx context.Context, operation hooks.Operation, id any, todo *models.Todo) error {
    if operation == hooks.OperationCreate {
        // Read user_id from context
        if userID := ctx.Value("user_id"); userID != nil {
            if uid, ok := userID.(string); ok {
                todo.UserId = &uid  // Populate server-side
            }
        }
    }
    return nil
}
```

### Security Model

This architecture provides multiple security layers:

1. **JWT Validation**: Only valid tokens pass the middleware
2. **Server-Side Population**: Fields are set from trusted JWT claims, not client input
3. **DTO Exclusion**: DTOs prevent clients from sending protected fields
4. **Context Isolation**: User info flows through secure context, not modifiable by client

### Adding Custom Context Values

You can extend this pattern for other use-cases:

```go
// Add custom context value in middleware
func CustomMiddleware(handler fiber.Handler) fiber.Handler {
    return func(c *fiber.Ctx) error {
        c.Locals("custom_key", "custom_value")
        return handler(c)
    }
}

// Access in ContextWithUser
func ContextWithUser(c *fiber.Ctx) context.Context {
    ctx := c.Context()

    if user := GetAuthenticatedUser(c); user != nil {
        ctx = context.WithValue(ctx, "user_id", user.UserID)
    }

    // Add custom values
    if custom := c.Locals("custom_key"); custom != nil {
        ctx = context.WithValue(ctx, "custom_key", custom)
    }

    return ctx
}
```

---

## 🤝 Contributing

We welcome contributions from developers of all experience levels! Whether you're fixing bugs, adding features, improving documentation, or sharing ideas, your input helps make **gorest** better for everyone.

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
- Bring a new feature to the table, let's discuss it arround a merge request

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
- Improve CLI
- Add progress indicators for generators
- Create interactive setup wizard

### 📝 Contribution Guidelines

- **Code Style**: Follow standard Go conventions (use `gofmt`, `golint`)
- **Commit Messages**: Use [conventional commits](https://www.conventionalcommits.org/) format
  - `feat:` for new features
  - `fix:` for bug fixes
  - `docs:` for documentation
  - `test:` for tests
  - `refactor:` for code improvements
  - `chore:` for chore tasks
- **Testing**: All new code behavior should include tests
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
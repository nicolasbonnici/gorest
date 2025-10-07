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

### 2. Start Test Database
```bash
make test-up
make test-schema
```

### 3. Generate Code
```bash
make test-generate
```

This generates:
- `gen/models/*.go` - Type-safe model structs
- `gen/resources/*.go` - REST API endpoints
- `gen/api/*.go` - OpenAPI schema stubs

### 4. Build & Run
```bash
make build
./bin/gorest
```

API available at: **http://localhost:3000**

---

## 📂 Project Structure
```
gorest/
├── cmd/gorest/main.go        # API server entrypoint
├── test/
│   ├── generate/main.go      # Code generator entrypoint
│   └── sql/schema.sql        # Test database schema
├── internal/                 # Core logic
│   ├── modelgen.go           # Model generator
│   ├── apigen.go             # REST API generator
│   ├── auth.go               # JWT authentication
│   ├── openapi.go            # OpenAPI spec generator
│   ├── utils.go              # Shared utilities
│   └── crud/                 # Generic CRUD operations
│       ├── crud.go           # Type-safe CRUD implementation
│       └── model.go          # Model interface
├── gen/                      # Generated code (gitignored)
│   ├── models/               # Database models
│   ├── resources/            # REST endpoints
│   └── api/                  # OpenAPI schema stubs
├── Makefile
├── compose.yml
└── .github/workflows/        # CI/CD
```

---

## 🛠 Development Commands

```bash
make test-up          # Start test database
make test-schema      # Load database schema
make test-generate    # Generate models & API
make build            # Build binary
make test             # Run all tests
```

---

## 📚 How It Works

1. **Schema Introspection**: Reads PostgreSQL `information_schema`
2. **Model Generation**: Creates Go structs with proper types & tags
3. **API Generation**: REST endpoints with Fiber handlers using generic CRUD
4. **Build**: Compile everything into a single binary

---

## 📜 License
MIT – free to use in your projects 🚀
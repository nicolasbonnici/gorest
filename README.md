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
- `gen/crud/crud.go` - Generic CRUD operations
- `gen/resources/*.go` - REST API endpoints

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
│   ├── generate/main.go      # Code generator
│   └── sql/schema.sql        # Test database schema
├── internal/                 # Code generation logic
│   ├── modelgen.go           # Model & CRUD generator
│   ├── apigen.go             # REST API generator
│   ├── auth.go               # JWT authentication
│   └── openapi.go            # OpenAPI spec generator
├── gen/                      # Generated code (gitignored)
│   ├── models/               # Database models
│   ├── crud/                 # CRUD operations
│   └── resources/            # REST endpoints
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
3. **CRUD Generation**: Generic type-safe CRUD operations
4. **API Generation**: REST endpoints with Fiber handlers
5. **Build**: Compile everything into a single binary

---

## 📜 License
MIT – free to use in your projects 🚀
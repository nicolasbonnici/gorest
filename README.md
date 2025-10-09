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
├── pkg/
│   ├── gorest/main.go        # API server entrypoint
│   └── gorest.go             # Core package
├── test/
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
├── .env.example              # Environment variables template
├── Makefile
├── compose.yml
└── .github/workflows/        # CI/CD
```

---

## 🛠 Development Commands

```bash
make test-up          # Start test database
make test-schema      # Load database schema
make build            # Build binary
make test             # Run all tests
```

---

## 📚 How It Works

1. **Schema Introspection**: Reads PostgreSQL `information_schema` at startup
2. **Code Generation**: Creates models, API endpoints, and OpenAPI specs on-the-fly
3. **API Server**: Serves REST endpoints with Fiber handlers using generic CRUD
4. **Runtime**: All code generation and API serving happen in a single binary

---

## 📜 License
MIT – free to use in your projects 🚀
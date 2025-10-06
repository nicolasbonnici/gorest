# gorest

🚀 **gorest** is a generic REST API generator for PostgreSQL written in Go.  
It introspects your database tables and relationships to expose **CRUD endpoints automatically**.

## ✨ Features
- 🔎 Auto-discovery of tables & columns
- 🔗 Auto-detection of **foreign key relationships**
- 🛠 CRUD endpoints for each table
- 📂 Nested sub-resources (`/users/:id/orders`)
- 🧩 Query expansion with `?expand=parent`
- 🔑 JWT authentication (basic)
- 📜 OpenAPI 3.0 spec (`/openapi.json`)
- 🐳 Docker support

---

## ⚙️ Requirements
- Go **1.23+**
- Docker & Docker Compose
- PostgreSQL **18+**

---

## 🚀 Getting Started

### 1. Clone the project
```bash
git clone https://github.com/nicolasbonnici/gorest.git
cd gorest
```

### 2. Run locally (without Docker)

#### Start PostgreSQL (local or Docker)
```bash
docker run --name gorest_db   -e POSTGRES_USER=postgres   -e POSTGRES_PASSWORD=postgres   -e POSTGRES_DB=mydb   -p 5432:5432   -d postgres:15
```

#### Create tables
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    firstname TEXT NOT NULL,
    lastname TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password TEXT,
    updated_at TIMESTAMP(0) WITHOUT TIME ZONE,
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_email ON users (email);

CREATE TABLE todo (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    user_id UUID REFERENCES users(id),
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    updated_at TIMESTAMP(0) WITHOUT TIME ZONE,
    created_at TIMESTAMP(0) WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_todo_title ON todo (title);
```

#### Run the API
```bash
go mod tidy
go run ./cmd/server
```

👉 API available at:
- http://localhost:3000
- http://localhost:3000/openapi.json

---

### 3. Run with Docker Compose
```bash
docker-compose up --build
```

Services:
- **API** → http://localhost:3000
- **Postgres** → `postgres://postgres:postgres@localhost:5432/mydb`

---

## 🔐 Authentication

Get a token:
```bash
curl -X POST http://localhost:3000/login   -H "Content-Type: application/json"   -d '{"username":"admin","password":"password"}'
```

Response:
```json
{"token":"<JWT_TOKEN>"}
```

Use the token:
```bash
curl -H "Authorization: Bearer <JWT_TOKEN>" http://localhost:3000/users
```

---

## 📚 Example API Usage

### ➕ Create a user
```bash
curl -X POST http://localhost:3000/users   -H "Content-Type: application/json"   -d '{"name":"Alice","email":"alice@test.com"}'
```

### ➕ Create an order linked to a user
```bash
curl -X POST http://localhost:3000/orders   -H "Content-Type: application/json"   -d '{"user_id":"<UUID_USER>","amount":120.5}'
```

### 📋 List all orders for a user
```bash
curl http://localhost:3000/users/<UUID_USER>/orders
```

### 🔎 Expand relations
```bash
curl http://localhost:3000/orders?expand=users
```

---

## 📜 OpenAPI
OpenAPI spec is available at:
```
http://localhost:3000/openapi.json
```
You can import it into **Swagger UI** or **Postman**.

---

## 📂 Project Structure
```
gorest/
├── cmd/server/main.go        # Entrypoint
├── internal/                 # Internal logic
│   ├── api.go                # CRUD + relations
│   ├── auth.go               # JWT authentication
│   ├── db.go                 # Schema introspection
│   └── openapi.go            # OpenAPI generator
├── pkg/gorest.go            # API bootstrap
├── Dockerfile
├── compose.yml
├── go.mod
└── go.sum
```

---

## 🛠 Development

Install dependencies:
```bash
make tidy
```

Run locally:
```bash
make run
```

Build binary:
```bash
make build
```

Run tests:
```bash
make test
```

---

## 📜 License
MIT – free to use in your projects 🚀
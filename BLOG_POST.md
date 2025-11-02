![GoREST - Generate Production-Ready REST APIs in Seconds](https://dev-to-uploads.s3.amazonaws.com/uploads/articles/032ymodam3bqnuf8fz0r.png)

Tired of writing CRUD endpoints by hand? Copy-pasting Go structs? Spending hours on Swagger documentation? Reimplementing the same authentication flow for the hundredth time? **GoREST solves all of that.**

With his first release GoREST offers a production-grade code generator that:
- 🔎 Auto-discovery of tables, relations, columns & types
- ⚡ Type-safe generic CRUD operations with hooks system
- 🛠 Scaffold REST endpoints for each table
- 🔐 Full DTO support with field-level control (`dto` tags)
- 🔑 JWT authentication with context-aware middleware
- 🎭 Auto-population of fields from authentication context
- 🌐 JSON-LD support with semantic web context (@context, @type, @id)
- 🔗 Automatic foreign key to IRI conversion (e.g., `/users/{uuid}`)
- 🔍 Advanced filtering & ordering (equality, comparison, text search, multiple fields)
- 📄 Page-based pagination with Hydra collections
- 👨🏻‍💻 DAL for PostgreSQL, MySQL and SQLite engines
- 🛡️ Production grade errors and processes management
- 🐳 Docker support with multi-database testing
- 🧪 Full test coverage with automated testing
- 💚 Health check endpoint (`/health`)
- 📜 OpenAPI 3.0 spec generation
---

## 🎯 Why GoREST?

GoREST isn't just another CRUD generator. It's a complete solution for building production-ready REST APIs:

| **Feature** | **GoREST** | **Manual Implementation** |
|-------------|-----------|---------------------------|
| **Time to API** | 2 commands (~30 seconds) | Days/weeks |
| **Schema Discovery** | Auto-discovery of tables, relations, columns & types | Manual schema mapping |
| **Database Support** | PostgreSQL, MySQL, SQLite (DAL) | Pick one, code for one |
| **Type Safety** | Auto-generated models with type-safe CRUD | Manual struct definitions |
| **REST Endpoints** | Scaffold for each table | Write every endpoint |
| **OpenAPI Docs** | Auto-generated 3.0 spec | Manual Swagger annotations |
| **Authentication** | Built-in JWT + context-aware middleware | Implement from scratch |
| **Field Auto-Population** | Auto-populate from auth context | Manual implementation |
| **DTOs** | Auto-generated with field-level security (`dto` tags) | Manual for every endpoint |
| **JSON-LD Support** | Built-in semantic web (@context, @type, @id) | Complex manual implementation |
| **Foreign Key IRIs** | Automatic conversion (e.g., `/users/{uuid}`) | Manual hypermedia links |
| **Filtering** | Equality, comparison, text search, multiple fields | Hours of coding |
| **Ordering** | Multi-field sorting out of the box | Manual SQL ORDER BY logic |
| **Pagination** | Hydra collections with page-based navigation | Manual cursor/offset implementation |
| **Hooks System** | StateProcessor, SQLQueryOverride, Serializer | Custom middleware for each case |
| **Production Features** | Health checks, graceful shutdown, structured logging | One-by-one implementation |
| **Error Handling** | Production-grade error management | Custom error handling |
| **Testing** | Full test coverage with Docker support | Write all tests manually |

---

## 📌 Real-World Example: Todo API in 60 Seconds

![Another One DJ Khaled meme](https://dev-to-uploads.s3.amazonaws.com/uploads/articles/kysfu3ptmlghh7bidv8t.png)

### 1. Start with Your Existing Database

No need to change your schema. GoREST works with what you have:

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    firstname TEXT NOT NULL,
    lastname TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password TEXT,
    created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE todo (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### 2. Generate Everything (Models, DTOs, Routes, OpenAPI)

```bash
git clone https://github.com/nicolasbonnici/gorest.git
cd gorest
cp .env.example .env
# Edit .env with your database connection

make generate     # Generates models, DTOs, resources, routes, OpenAPI spec
make run         # API is live!
```

**That's it.** Your API is running at **http://localhost:3000** with:

- ✅ **Full CRUD endpoints** for todos and users
- ✅ **OpenAPI documentation** at `/openapi.json`
- ✅ **Health check** at `/health`
- ✅ **JWT authentication** ready to use
- ✅ **Type-safe DTOs** with automatic validation
- ✅ **JSON-LD support** for semantic web compatibility

**Generated Endpoints:**

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/todos` | List todos (with filtering, ordering, pagination) |
| `POST` | `/todos` | Create todo |
| `GET` | `/todos/:id` | Get specific todo |
| `PUT` | `/todos/:id` | Update todo |
| `DELETE` | `/todos/:id` | Delete todo |
| `GET` | `/users` | List users |
| `POST` | `/users` | Create user |
| `GET` | `/users/:id` | Get user |
| `PUT` | `/users/:id` | Update user |
| `DELETE` | `/users/:id` | Delete user |

---

## 🔐 Security Done Right: DTOs & Field Control

Here's where GoREST shines. Ever had a client send `user_id` in a request to access someone else's data? Not anymore.

### Automatic DTO Generation

GoREST generates three DTO types for every model:

1. **CreateDTO** - For POST requests (excludes auto-generated fields like `id`, `created_at`)
2. **UpdateDTO** - For PUT requests (excludes auto-generated and protected fields)
3. **ResponseDTO** - For GET responses (shows everything the client should see)

### Field-Level Control with `dto` Tags

Control what clients can send and receive using model tags:

```go
type Todo struct {
    Id        string     `json:"id" db:"id"`
    UserId    *string    `json:"user_id" db:"user_id" dto:"read"`  // 🔒 Read-only!
    Title     string     `json:"title" db:"title"`
    Content   string     `json:"content" db:"content"`
    CreatedAt *time.Time `json:"created_at" db:"created_at"`
}
```

**Result:** Clients **cannot** send `user_id` in POST/PUT requests. GoREST populates it server-side from JWT authentication:

```go
// Client sends (TodoCreateDTO):
{
  "title": "Buy groceries",
  "content": "Milk, eggs, bread"
  // No user_id - field not in DTO
}

// Server populates user_id from JWT token (happens in hooks)
// Client receives (TodoDTO):
{
  "id": "abc-123",
  "user_id": "xyz-789",  // ✅ Populated server-side
  "title": "Buy groceries",
  "content": "Milk, eggs, bread",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Security guarantee:** Client cannot spoof `user_id`. It's extracted from the JWT token by the server. Period.

---

## 🔍 Advanced Filtering, Ordering & Pagination

GoREST provides production-grade querying out of the box. No code needed - it's all generated for you.

### Filtering

Filter your data using intuitive query parameters with support for various operators:

```bash
# Equality
GET /todos?status=active

# Multiple values (OR)
GET /todos?status[]=active&status[]=pending

# Comparison operators
GET /todos?priority[gte]=5           # Greater than or equal
GET /todos?priority[lte]=10          # Less than or equal
GET /todos?priority[gt]=3            # Greater than
GET /todos?priority[lt]=8            # Less than
GET /todos?priority[ne]=0            # Not equal

# Text search
GET /todos?title[like]=meeting       # Case-sensitive
GET /todos?title[ilike]=MEETING      # Case-insensitive

# Date filtering
GET /todos?created_at[gte]=2024-01-01
GET /todos?updated_at[lt]=2024-12-31

# Combine filters (AND)
GET /todos?status=active&priority[gte]=7&created_at[gte]=2024-01-01
```

### Ordering

Sort results by one or multiple fields:

```bash
# Single field
GET /todos?order[priority]=desc
GET /todos?order[created_at]=asc

# Multiple fields (order matters)# GoREST: The Code Generator That Turns Your Database Into a Production-Ready REST API
GET /todos?order[priority]=desc&order[created_at]=asc
```

The first `order` parameter becomes the primary sort, second becomes secondary, etc.

### Pagination with Hydra Collections

GoREST uses **Hydra** for semantic pagination:

```bash
GET /todos?page=2&limit=20
```

**Response:**
```json
{
  "@context": "http://www.w3.org/ns/hydra/context.jsonld",
  "@type": "hydra:Collection",
  "hydra:totalItems": 42,
  "hydra:member": [
    {
      "@id": "/todos/abc-123",
      "@type": "TodoDTO",
      "id": "abc-123",
      "title": "Buy groceries",
      "user_id": "/users/xyz-789"
    }
  ],
  "hydra:view": {
    "@type": "hydra:PartialCollectionView",
    "hydra:first": "/todos",
    "hydra:previous": "/todos",
    "hydra:next": "/todos?page=3",
    "hydra:last": "/todos?page=5"
  }
}
```

### Real-World Combined Query

```bash
GET /todos?status[]=active&status[]=pending&priority[gte]=5&order[priority]=desc&order[created_at]=desc&page=2&limit=20
```

This single query:
- ✅ Filters todos with status "active" OR "pending"
- ✅ AND priority >= 5
- ✅ Sorts by priority (descending), then created_at (descending)
- ✅ Returns page 2 with 20 items per page
- ✅ Includes navigation links (first, previous, next, last)

**All of this is auto-generated.** Zero code required.

### Security

Filtering and ordering are secure by design:
- 🔒 Only database fields (with `db` tags) can be filtered/sorted
- 🔒 SQL injection protection via parameterized queries
- 🔒 Invalid fields are silently ignored
- 🔒 No arbitrary SQL execution possible

---

## 🪝 The Hooks System: Customize Without Breaking Generated Code

![SpongeBob Magic meme](https://dev-to-uploads.s3.amazonaws.com/uploads/articles/pz0znd6p7zm0d6idmcp7.png)

GoREST generates code, but **you control the business logic** through hooks. Here's the real power:

### Hook Types

1. **StateProcessor** - Validate/transform data before Create/Update/Delete
2. **SQLQueryOverride** - Replace default queries with custom SQL
3. **SQLQueryListener** - Intercept queries before/after execution
4. **Serializer** - Transform data before API response

### Example 1: Auto-Populate User ID (Server-Side Security)

```go
type TodoHooks struct {
    hooks.NoOpHooks[models.Todo]
}

func (h *TodoHooks) StateProcessor(ctx context.Context, operation hooks.Operation, id any, todo *models.Todo) error {
    if operation == hooks.OperationCreate {
        // Validate
        if len(todo.Title) < 3 {
            return fmt.Errorf("title must be at least 3 characters")
        }

        // Auto-populate user_id from JWT context (server-side!)
        if userID := ctx.Value("user_id"); userID != nil {
            if uid, ok := userID.(string); ok {
                todo.UserId = &uid
            }
        }
    }
    return nil
}
```

### Example 2: Custom SQL Queries

```go
func (h *TodoHooks) SQLQueryOverride(ctx context.Context, operation hooks.Operation, id any) (*sqlbuilder.SelectQuery, error) {
    if operation == hooks.OperationList {
        // Only show incomplete todos
        query := sqlbuilder.
            Select("id", "title", "content", "user_id").
            From("todo").
            Where("completed = FALSE")
        return query, nil
    }
    return nil, nil
}
```

### Example 3: Per-Resource Authorization

```go
func (h *TodoHooks) Authorize(ctx context.Context, operation hooks.Operation, id any) error {
    userID := ctx.Value("user_id")
    if userID == nil {
        return fmt.Errorf("authentication required")
    }

    // Add custom role checks, permissions, etc.
    if operation == hooks.OperationDelete {
        // Only admins can delete
        if !isAdmin(userID) {
            return fmt.Errorf("insufficient permissions")
        }
    }

    return nil
}
```

**Key Benefit:** Hooks live in separate files. Regenerate your API anytime without losing business logic.

---

## ⚡ Performance: Fast Enough to Make You Smile

Local benchmarks on Framework 14" (i7 12th gen, 16GB RAM):

| **Resources** | **Avg Time** | **Max Time** | **Format** |
|---------------|--------------|--------------|------------|
| 1 | 0.2ms | 0.5ms | JSON-LD |
| 10 | 0.4ms | 0.8ms | JSON-LD |
| 100 | **0.9ms** | **1.4ms** | JSON-LD |
| 1,000 | 8ms | 12ms | JSON-LD |
| 10,000 | 75ms | 90ms | JSON-LD |

**Highlights:**
- ⚡ **100 resources in <1ms** (including JSON-LD transformation)
- 🚀 **Sub-millisecond response times** for typical queries
- 📦 **JSON-LD overhead is negligible** thanks to efficient formatters

Run your own benchmarks: `make benchmark`

---

## 🌐 JSON-LD: Semantic Web Out of the Box

GoREST automatically supports **JSON-LD** (Linked Data), making your API:
- 🤖 **Machine-readable** - semantic context for AI/ML tools
- 🔗 **Interoperable** - works with semantic web technologies
- 📈 **SEO-friendly** - structured data for search engines

**Regular JSON** vs **JSON-LD:**

```json
// Accept: application/json
{
  "id": "abc-123",
  "user_id": "xyz-789",
  "title": "Buy groceries"
}

// Accept: application/ld+json
{
  "@context": "https://schema.org/",
  "@type": "TodoDTO",
  "@id": "/todos/abc-123",
  "id": "abc-123",
  "user_id": "/users/xyz-789",  // 🔗 Auto-converted to IRI
  "title": "Buy groceries"
}
```

**Foreign keys automatically become IRIs:**
- `user_id: "xyz-789"` → `user_id: "/users/xyz-789"`
- Perfect for hypermedia-driven APIs (HATEOAS)
- Zero configuration required

---

## 🛡️ Production Features

### Health Checks

```bash
curl http://localhost:3000/health
```

```json
{
  "status": "healthy",
  "database": {
    "status": "up"
  }
}
```

Perfect for:
- Load balancer health checks
- Kubernetes readiness/liveness probes
- Monitoring systems (Prometheus, Datadog)

### Graceful Shutdown

- Listens for `SIGTERM` and `SIGINT`
- 30-second timeout for in-flight requests
- Cleanly closes database connections
- Zero data corruption risk

### Structured Logging

```go
import "github.com/nicolasbonnici/gorest/internal/logger"

logger.Log.Info("User created", "user_id", userID, "email", email)
logger.Log.Error("Database error", "error", err, "query", query)
```

JSON output perfect for log aggregation (ELK, Splunk, etc.)

---

## 🤝 Contribute & Help Shape GoREST

GoREST is **100% open-source** and we need **YOU** to make it better!

### 🌟 Why Contribute?

- **Learn Go, SQL, code generation, REST APIs** in a real-world project
- **Your code helps thousands of developers** build better APIs faster
- **Build your portfolio** with production-grade open-source contributions
- **Join a welcoming community** of passionate developers
- **Support the free software movement**

### 💡 How to Contribute

#### **1. Test & Report Bugs**
```bash
git clone https://github.com/nicolasbonnici/gorest.git
cd gorest
make test-up && make test-schema
make generate && make run
```

Found a bug? [Open an issue](https://github.com/nicolasbonnici/gorest/issues)!

#### **2. Contribute Code**

**For Beginners:**
- Fix typos in documentation
- Add code comments
- Write examples and tutorials
- Improve error messages

**Intermediate:**
- Add tests for existing features
- Fix bugs from the issue tracker
- Improve code generation templates
- Add validation helpers

**Advanced:**
- Add support for new databases (MariaDB, CockroachDB?)
- Implement GraphQL support alongside REST
- Add gRPC code generation
- Create monitoring/metrics integrations
- Build a web UI for API management

#### **3. Improve Documentation**

- Write blog posts about your GoREST experience
- Create video tutorials
- Translate documentation to other languages
- Add architecture diagrams
- Write migration guides

#### **4. Share Ideas**

Have a feature idea? Open a GitHub Discussion or Issue!

### 🚀 Easy First Contributions

1. **Add Database Support**: GoREST supports PostgreSQL, MySQL, SQLite - what about MariaDB or CockroachDB?
2. **Write Hook Examples**: Share your real-world hook implementations
3. **Create Middleware**: Rate limiting, CORS, request logging, etc.
4. **Improve Benchmarks**: Test with different databases and workloads
5. **Build Integrations**: Prometheus metrics, OpenTelemetry, etc.

### 📋 Contribution Guidelines

- **Fork the repo** and create a feature branch
- **Write tests** for new features
- **Follow Go conventions** (gofmt, golint)
- **Use conventional commits** (`feat:`, `fix:`, `docs:`, etc.)
- **Update documentation** for user-facing changes
- **Open a PR** with a clear description

See [CONTRIBUTING.md](https://github.com/nicolasbonnici/gorest/blob/main/CONTRIBUTING.md) for detailed guidelines.

---

## 🎯 What Could YOU Build with GoREST?

- 💼 **SaaS Backend** - Multi-tenant APIs in minutes
- 📱 **Mobile App Backend** - REST APIs ready for iOS/Android
- 🤖 **Microservices** - Generate APIs for each service
- 🧪 **Prototypes** - Validate ideas fast
- 📊 **Data APIs** - Expose your database as a service
- 🎮 **Game Backends** - Player data, leaderboards, etc.

**What will you build?** Tell us in the comments!

---

## 🔗 Links & Resources

- **GitHub**: [github.com/nicolasbonnici/gorest](https://github.com/nicolasbonnici/gorest)
- **Documentation**: Full README with detailed guides
- **Issues**: [Report bugs or request features](https://github.com/nicolasbonnici/gorest/issues)
- **Contributing**: [CONTRIBUTING.md](https://github.com/nicolasbonnici/gorest/blob/main/CONTRIBUTING.md)

---

## ⭐ Support the Project

- **Star the repo** - it helps others discover GoREST
- **Share this article** - Twitter, LinkedIn, Reddit, Dev.to
- **Write about your experience** - blog posts, tweets, videos
- **Contribute code or docs** - every bit helps!

---

## 💬 Join the Conversation

**Have you tried GoREST?**
- What's your use case?
- What features would you like to see?
- Want to contribute but don't know where to start?

**Drop a comment below!** We reply to everyone.

**Ready to generate your first API?**

```bash
git clone https://github.com/nicolasbonnici/gorest.git
cd gorest
make generate && make run
```

**It's that simple.** 🚀

---

*GoREST is MIT licensed and free to use in your projects. Built with Go, for developers who value their time.*

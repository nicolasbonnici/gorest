# Basic API Example

This example demonstrates how to use GoREST to build a REST API from your database schema.

## Development vs Production

**For development** (within the GoREST repository):
- This example uses `replace github.com/nicolasbonnici/gorest => ../..` in `go.mod`
- Allows testing unreleased changes locally

**For production** (standalone projects):
- Use `go.mod.production` as reference
- Import the published module: `github.com/nicolasbonnici/gorest v0.2.0`
- No replace directive needed

## Setup

1. **Create your project structure**:
```bash
mkdir my-api && cd my-api
go mod init github.com/yourusername/my-api
```

2. **Install GoREST**:
```bash
go get github.com/nicolasbonnici/gorest
```

3. **Configure environment variables**:
```bash
# Initialize .env from template
make init

# Edit .env with your settings
nano .env
```

Or manually:
```bash
cp .env.dist .env
nano .env
```

Minimum required variables:
```bash
DATABASE_URL=postgres://user:password@localhost:5432/mydb?sslmode=disable
JWT_SECRET=your-super-secret-jwt-key-min-32-chars
```

Optional variables (defaults from `gorest.yaml` will be used if not set):
```bash
PORT=3000
ENVIRONMENT=development
PAGINATION_LIMIT=10
PAGINATION_MAX_LIMIT=1000
CORS_ORIGINS=*
```

4. **Create `gorest.yaml`** to configure code generation:
```yaml
codegen:
  output:
    models: "generated/models"
    resources: "generated/resources"
    dtos: "generated/dtos"
    openapi: "generated/openapi"
    config: "generated/config"
  auth:
    enabled: true
  endpoints:
    - name: users
      auth:
        GET: true
        POST: true
        PUT: true
        DELETE: true

server:
  port: 3000
  environment: "development"

database:
  url: "${DATABASE_URL}"

pagination:
  default_limit: 10
  max_limit: 1000

auth_defaults:
  GET: true
  POST: true
  PUT: true
  DELETE: true

plugins:
  - name: auth
    enabled: true
    config:
      jwt_secret: "${JWT_SECRET}"
      jwt_ttl: 900
  # ... other plugins
```

5. **Generate code from your database**:
```bash
# Generate all (models, resources, DTOs, OpenAPI)
go run github.com/nicolasbonnici/gorest/cmd/codegen@latest all

# Or use the Makefile
make generate
```

6. **Create your main.go** (see example in this directory)

7. **Run your API**:
```bash
go run main.go
```

## Docker Setup

This example includes a Docker Compose configuration for running the API with PostgreSQL:

```bash
# Set environment variables
export JWT_SECRET="your-secret-key-minimum-32-characters-long"
export DB_PASSWORD="postgres"

# Start database and API
docker compose up

# Or build and run in detached mode
docker compose up -d --build
```

The API will be available at http://localhost:3000.

To stop:
```bash
docker compose down
```

## Project Structure

After running the generators, your project will look like:

```
my-api/
├── go.mod
├── go.sum
├── .env
├── gorest.yaml (optional)
├── main.go
├── models/           # Generated model structs
│   ├── user.go
│   └── ...
├── resources/        # Generated REST handlers
│   ├── user.go
│   ├── routes.go    # Auto-generated route registration
│   └── ...
├── dtos/            # Generated DTOs
│   ├── user.go
│   └── ...
├── openapi/         # Generated OpenAPI spec
│   └── openapi_gen.go
└── config/          # Configuration files
    └── auth.json    # Generated auth config
```

## Key Features

- **Automatic CRUD**: All models get full REST endpoints (GET, POST, PUT, DELETE)
- **Authentication**: JWT-based auth with bcrypt password hashing
- **Pagination**: Built-in pagination with configurable limits
- **Filtering**: Advanced query filters on all resources
- **OpenAPI**: Auto-generated OpenAPI 3.0 spec at `/openapi.json`
- **Interactive Docs**: Swagger UI at `/openapi`
- **Health Check**: Health endpoint at `/health`

## API Endpoints

Once running, your API will have:

- `POST /auth/register` - Register new user
- `POST /auth/login` - Login and get JWT token
- `GET /health` - Health check
- `GET /openapi` - Interactive API documentation
- `GET /openapi.json` - OpenAPI 3.0 specification
- `GET /{resource}` - List resources (with pagination, filtering, sorting)
- `GET /{resource}/{id}` - Get single resource
- `POST /{resource}` - Create resource
- `PUT /{resource}/{id}` - Update resource
- `DELETE /{resource}/{id}` - Delete resource

## Regenerating Code

Whenever you change your database schema:

```bash
# Using the Makefile
make generate

# Or directly with the codegen tool
go run github.com/nicolasbonnici/gorest/cmd/codegen@latest all

# Or individual steps
go run github.com/nicolasbonnici/gorest/cmd/codegen@latest models
go run github.com/nicolasbonnici/gorest/cmd/codegen@latest resources
go run github.com/nicolasbonnici/gorest/cmd/codegen@latest openapi
```

## Custom Configuration

You can customize output directories and authentication per-resource in `gorest.yaml`:

```yaml
codegen:
  output:
    models: "internal/domain"      # Put models in a different location
    resources: "internal/api"      # Custom API location
    dtos: "internal/api/dto"       # Custom DTO location
    openapi: "docs/openapi"        # Custom OpenAPI location
    config: "config"               # Config directory
  auth:
    enabled: true
  endpoints:
    - name: users
      auth:
        GET: true
        POST: true
        PUT: true
        DELETE: true
    - name: posts
      auth:
        GET: false    # Public read
        POST: true    # Auth required
        PUT: true
        DELETE: true

auth_defaults:
  GET: true
  POST: true
  PUT: true
  DELETE: true
```

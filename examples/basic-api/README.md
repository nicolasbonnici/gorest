# Basic API Example

This example demonstrates how to use GoREST to build a REST API from your database schema.

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

3. **Create environment file** (`.env`):
```bash
DATABASE_URL=postgres://user:password@localhost:5432/mydb?sslmode=disable
JWT_SECRET=your-super-secret-jwt-key-min-32-chars
JWT_TTL=900
PORT=3000
PAGINATION_LIMIT=100
PAGINATION_MAX_LIMIT=1000
CORS_ORIGINS=http://localhost:3000
```

4. **Optional: Create `gorest.yaml`** to customize output directories:
```yaml
output:
  models: "models"
  resources: "resources"
  dtos: "dtos"
  openapi: "openapi"
  config: "config"
```

If you don't create this file, GoREST will use the defaults shown above.

5. **Generate code from your database**:
```bash
# Generate models
go run github.com/nicolasbonnici/gorest/cmd/modelgen@latest

# Generate REST resources (interactive)
go run github.com/nicolasbonnici/gorest/cmd/resourcegen@latest

# Generate OpenAPI spec
go run github.com/nicolasbonnici/gorest/cmd/openapigen@latest
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
# Regenerate models
go run github.com/nicolasbonnici/gorest/cmd/modelgen@latest

# Regenerate resources (will prompt for overwrites)
go run github.com/nicolasbonnici/gorest/cmd/resourcegen@latest

# Regenerate OpenAPI
go run github.com/nicolasbonnici/gorest/cmd/openapigen@latest
```

## Custom Configuration

You can customize output directories by creating a `gorest.yaml` file:

```yaml
output:
  models: "internal/domain"      # Put models in a different location
  resources: "internal/api"      # Custom API location
  dtos: "internal/api/dto"       # Custom DTO location
  openapi: "docs/openapi"        # Custom OpenAPI location
  config: "config"               # Config directory
```

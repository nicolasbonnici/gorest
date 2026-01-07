# Configuration

GoREST uses `gorest.yaml` for all configuration. The file has four main sections:

## Code Generation (`codegen`)

Controls how code is generated from your database:

```yaml
server:
  scheme: "${SERVER_SCHEME:-http}"
  host: "${SERVER_HOST:-localhost}"
  port: "${SERVER_PORT:-8000}"
  environment: "${ENV:-development}"

database:
  url: "${DATABASE_URL}"

pagination:
  default_limit: "${PAGINATION_DEFAULT_LIMIT:-10}"
  max_limit: "${PAGINATION_MAX_LIMIT:-1000}"

plugins:
  - name: auth
    enabled: true
    config:
      jwt_secret: "${JWT_SECRET}"
      jwt_ttl: 900

codegen:
  output:
    models: "generated/models"
    resources: "generated/resources"
    dtos: "generated/dtos"
    openapi: "generated/openapi"
    config: "generated/config"

  enums:
    enabled: true

  auth:
    enabled: true
    defaults:
      GET: true
      POST: true
      PUT: true
      DELETE: true
    endpoints:
      - name: posts
        GET: false
```

## Runtime Configuration (`server`, `database`, `pagination`)

Basic server settings:

```yaml
server:
  scheme: "${SERVER_SCHEME:-http}"
  host: "${SERVER_HOST:-localhost}"
  port: "${SERVER_PORT:-3000}"
  environment: "${ENV:-development}"

database:
  url: "${DATABASE_URL}"

pagination:
  default_limit: "${PAGINATION_DEFAULT_LIMIT:-10}"
  max_limit: "${PAGINATION_MAX_LIMIT:-1000}"
```

### Environment Variable Interpolation

GoREST supports bash-style environment variable interpolation with default fallback values in any string configuration value.

**Syntax:**
- `${VAR}` - Replace with environment variable VAR value (keeps `${VAR}` if not set)
- `${VAR:-default}` - Replace with VAR value, or use "default" if VAR is not set

**Examples:**
```yaml
# Simple defaults
server:
  port: "${PORT:-3000}"
  host: "${HOST:-localhost}"

# Complex URL with default
database:
  url: "${DATABASE_URL:-postgres://localhost:5432/dev?sslmode=disable}"

# Mixed usage - some required, some with defaults
server:
  scheme: "${SCHEME:-http}"     # Optional with default
  host: "${HOST}"               # Required (error if not set)
  port: "${PORT:-3000}"         # Optional with default

# Plugin configuration
plugins:
  - name: auth
    config:
      jwt_secret: "${JWT_SECRET}"                    # Required
      jwt_ttl: "${JWT_TTL:-900}"                     # Optional
      issuer: "${JWT_ISSUER:-gorest-api}"           # Optional

  - name: cors
    config:
      origins: "${CORS_ORIGINS:-http://localhost:3000}"
```

**Important Notes:**
- If a variable is set to an empty string (`VAR=""`), the empty string is used (not the default)
- Only use `os.LookupEnv()` to check existence, not just `os.Getenv()`
- Defaults can contain any characters including `:`, `=`, `/`, `?`, etc.
- Multiple variables can be used in the same value: `"${SCHEME:-http}://${HOST:-localhost}:${PORT:-3000}"`

## Plugin Configuration

All middleware and features are configured through plugins. Plugins are loaded in the order specified and must be manually applied to routes or route groups:

```yaml
server:
  cors_origins: "*"

plugins:
  - name: status
    enabled: true
  - name: auth
    enabled: true
    config:
      jwt_secret: "${JWT_SECRET}"
      jwt_ttl: 900
```

## Environment-Specific Overrides

Create `gorest.{environment}.yaml` files to override base config:

**gorest.production.yaml**:
```yaml
server:
  environment: "production"
  cors_origins: "https://app.example.com"

plugins:
  - name: ratelimit
    enabled: true
    config:
      requests_per_second: 50
      burst: 100
```

Load with `ENVIRONMENT` variable:
```bash
export ENVIRONMENT=production
```

## Template

See [`gorest.yaml.example`](gorest.yaml.example) for a complete documented template.

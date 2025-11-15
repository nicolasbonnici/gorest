# Building GoREST with Version Information

GoREST automatically includes version information in the `X-Powered-By` response header.

## Version Injection

The version is injected at build time using Go's `-ldflags`:

```bash
# Get version from git tag
VERSION=$(git describe --tags --always --dirty)

# Build with version
go build -ldflags="-X 'github.com/nicolasbonnici/gorest.Version=${VERSION}'"
```

## Version Format

- **Tagged releases**: `v0.2.0`
- **Development builds**: `v0.2.0-5-gabcd123` (5 commits after v0.2.0, commit abcd123)
- **Dirty builds**: `v0.2.0-dirty` (uncommitted changes)
- **No git**: `dev` (default fallback)

## Building Examples

### Basic API Example

```bash
cd examples/basic-api
make build
```

The Makefile automatically injects the version.

### Manual Build

```bash
# Development build (shows "dev")
go build -o myapi .

# Build with version from git
VERSION=$(git describe --tags --always --dirty)
go build -ldflags="-X 'github.com/nicolasbonnici/gorest.Version=${VERSION}'" -o myapi .
```

## Verifying Version

Check the version in response headers:

```bash
curl -I http://localhost:3000/health
# Look for: X-Powered-By: GoREST/v0.2.0
```

Or in server logs:

```bash
# Version is logged on startup
2025-11-13T10:00:00Z INFO REST API running port=3000 version=v0.2.0
```

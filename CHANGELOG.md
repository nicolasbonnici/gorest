# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2025-12-05

### Breaking Changes
- **Plugin Architecture**: Complete restructuring to modular plugin system
  - Moved codegen functionality from plugin to core `/generator` library
  - Removed codegen plugin directory entirely
  - All core features now available as optional plugins
- **Code Generation**:
  - Renamed `cmd/gorest-codegen` → `cmd/codegen`
  - Commands now inline in CLI (no longer using plugin pattern for codegen)
  - Benchmark command moved to separate CLI in benchmark plugin

### Added
- **Core Plugins System**:
  - Authentication plugin with JWT support
  - Content type negotiation plugin
  - Logger plugin with structured logging
  - Rate limiter plugin
  - Request ID plugin
  - Security plugin (OWASP ASVS L2 compliant)
  - CORS management plugin
  - Health check plugin
  - OpenAPI plugin
- **Benchmark Plugin**:
  - Standalone benchmark plugin with its own CLI (`/cmd/benchmark`)
  - Built-in performance testing tool
  - Dedicated test server for benchmarks
  - Support for multiple concurrency levels (1, 10, 50)
  - Support for different data sizes (10, 100, 1000 records)
  - Detailed performance metrics (p50, p95, p99 latencies)
  - Auto-cleanup and schema restoration
- **Plugin Features**:
  - Plugin auto-discovery via `init()` registration
  - Plugin configuration via `gorest.yaml`
  - Plugin factory pattern for instantiation
  - Command plugin support for CLI tools
  - Middleware plugin support for HTTP handlers

### Changed
- **Architecture**:
  - `/generator` is now a pure library (no CLI concerns)
  - Plugin registration through `pluginloader.RegisterPluginFactory()`
  - Simplified command routing with inline handlers
  - Improved modularity and flexibility
- **CLI Structure**:
  - `cmd/codegen/` - Core code generation CLI
  - `cmd/benchmark/` - Symlink to `plugins/benchmark/cmd/` for discoverability
- **Performance**:
  - Sub-millisecond response times for small payloads (871µs median)
  - Under 2ms for medium payloads (100 items)
  - ~11ms for large payloads (1000 items)
  - Zero errors across all load tests
  - Excellent scalability with minimal degradation under load

### Security
- **OWASP ASVS Level 2 Compliance**:
  - Complete security headers implementation
  - TRACE method disabled
  - Rate limiting support
  - JWT security
  - SQL injection prevention
  - XSS prevention
  - Method whitelisting
- **Security Score**: 10/10 (perfect)
- **Mozilla Observatory**: A+ grade
- **Production Ready**: All security tests passing (27/27)

### Fixed
- Test suite issues with undefined functions
- Plugin loader test signature mismatches
- OpenAPI function visibility (moved to plugin)

### Removed
- Codegen plugin directory (`plugins/codegen/`)
- Obsolete test files for moved functions (`gorest_test.go`)

## [0.2.0] - 2025-11-07

### Breaking Changes
- **Major restructuring**: Transformed from application-focused to library-first architecture
- All core packages moved from `internal/` to root level for public export
- Package imports changed:
  - `github.com/nicolasbonnici/gorest/pkg/database` → `github.com/nicolasbonnici/gorest/database`
  - `github.com/nicolasbonnici/gorest/internal/crud` → `github.com/nicolasbonnici/gorest/crud`
  - `github.com/nicolasbonnici/gorest/internal/hooks` → `github.com/nicolasbonnici/gorest/hooks`
  - `github.com/nicolasbonnici/gorest/internal/formatter` → `github.com/nicolasbonnici/gorest/formatter`
  - `github.com/nicolasbonnici/gorest/internal/middleware` → `github.com/nicolasbonnici/gorest/middleware`

### Added
- New exportable packages at root level:
  - `auth/` - Authentication and authorization functionality
  - `filter/` - Query filtering and ordering
  - `pagination/` - Pagination helpers and Hydra collections
  - `response/` - HTTP response formatting and validation
  - `generator/` - Code generation utilities
- Example application in `examples/basic-api/`

### Changed
- Main library entry point moved to root `gorest.go`
- Package structure optimized for library consumption
- All tests and imports updated to new package paths

## [0.1.0-RC] - 2025-10-19

### Initial Release

**gorest** is a PostgreSQL REST API code generator for Go that automatically generates type-safe CRUD endpoints from your database schema.

### Features

- **Schema Introspection**: Automatic discovery of tables, columns, and types from PostgreSQL
- **Model Generation**: Type-safe Go structs with proper tags
- **DTO Support**: Separate DTOs for Create, Update, and Response operations with field visibility control
- **Field Visibility Control**: Manual annotation system using `dto` struct tags
  - `dto:"-"` - Exclude field completely
  - `dto:"read"` - Only in response DTOs
  - `dto:"write"` - Only in create/update DTOs
  - `dto:"read,write"` or no tag - Include in all DTOs (default)
- **REST API Generation**: CRUD endpoints with Fiber handlers
- **Auto-Generated Routes**: Automatic route registration
- **JWT Authentication**: Decorator pattern for endpoint protection
- **OpenAPI 3.0**: Automatic API documentation generation
- **Hooks System**: Extensible business logic integration
- **Health Check**: `/health` endpoint with database connectivity check
- **Graceful Shutdown**: Proper SIGTERM/SIGINT handling with 30s timeout
- **Type-Safe CRUD**: Generic CRUD operations with PostgreSQL support
- **Content Negotiation**: JSON and JSON-LD response formats
- **Full Test Coverage**: Automated testing for all components

### Project Structure

```
gorest/
├── cmd/                      # CLI tools
│   ├── modelgen/            # Model generator
│   ├── resourcegen/         # Resource generator
│   └── openapigen/          # OpenAPI generator
├── pkg/gorest/              # API server
├── internal/                # Core logic
│   ├── api/                 # Generated code (models, DTOs, resources)
│   ├── crud/                # Generic CRUD operations
│   ├── hooks/               # Business logic hooks
│   ├── middleware/          # HTTP middleware
│   └── formatter/           # Response formatters
├── test/                    # Test utilities and schemas
└── config/                  # Configuration files
```

### Getting Started

1. Configure database connection in `.env`
2. Run `make test-up && make test-schema` to start test database
3. Run `make generate` to generate all code
4. Run `make build && ./bin/gorest` to start the API

See [README.md](README.md) for complete documentation.

---

## Contributors

- Nicolas Bonnici (@nicolasbonnici)

---

## Notes

This is the first release candidate. Please report any issues at:
https://github.com/nicolasbonnici/gorest/issues

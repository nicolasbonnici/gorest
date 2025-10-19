# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0-RC] - 2025-10-19

### Added
- **Full DTO Support**: Implemented separate DTOs for Create, Update, and Response operations
  - `CreateDTO`: Used for POST requests, excludes auto-generated fields (id, created_at, updated_at)
  - `UpdateDTO`: Used for PUT requests, excludes auto-generated fields
  - `ResponseDTO`: Used for all responses, excludes sensitive fields (passwords, tokens, secrets)
- **Sensitive Field Exclusion**: Automatic exclusion of sensitive fields from response DTOs
  - Auto-detects patterns: `password*`, `*token`, `*secret`, `*api_key`
  - Prevents accidental exposure of sensitive data
- **Health Check Endpoint**: Added `/health` endpoint with database connectivity check
  - Returns 200 OK when healthy
  - Returns 503 Service Unavailable when database is down
  - Includes detailed status information
- **Graceful Shutdown**: Implemented proper shutdown handling
  - Listens for SIGTERM and SIGINT signals
  - 30-second timeout for graceful shutdown
  - Cleanly closes database connections
- **Auto-Generated Route Registration**: Route registration now fully automated
  - No more manual updates to `routes.go`
  - Automatically generates route registration based on discovered models
  - Handles auth requirements per resource
- **Configurable Test Database**: Test DB URL now configurable via `DATABASE_URL_TEST` environment variable
- **Enhanced Error Handling**:
  - Added logging for type casting failures in CRUD operations
  - Added database connection validation in generators (ping test)
  - Improved bounds checking in Accept header parsing
- **Code Organization**: Split large `apigen.go` (650 LOC) into 4 focused files:
  - `apigen.go`: Main orchestration (58 LOC)
  - `apigen_ast.go`: AST parsing functions
  - `apigen_dto.go`: DTO generation logic
  - `apigen_resource.go`: Resource generation logic
  - `apigen_types.go`: Type conversion utilities
- **Constants**: Created `constants.go` for magic strings and configuration values
  - Field name constants (id, created_at, updated_at)
  - Sensitive field patterns
  - Table name constants
  - Pagination constants

### Changed
- **Breaking**: API responses now use DTOs instead of raw models
  - Sensitive fields (passwords) are automatically excluded from responses
  - Response structure remains compatible for non-sensitive fields
- **Breaking**: POST requests now accept CreateDTO (excludes id, timestamps)
- **Breaking**: PUT requests now accept UpdateDTO (excludes id, timestamps)
- **Server Startup**: Now displays health check endpoint URL on startup
- **Resource Generation**: Resources now include automatic model ↔ DTO conversion functions
- **Route Registration**: Moved from hardcoded switch statement to auto-generated code
- **Test Configuration**: Test database URL defaults to standard test URL if env var not set
- **Table References**: Auth queries now use constants instead of hardcoded table names

### Removed
- **Dead Code**: Removed unused `JWTMiddleware` function (internal/auth.go:66-82)
  - Replaced by `RequireAuth` decorator pattern
- **Hardcoded Route Switch**: Removed manual route registration switch statement

### Fixed
- Type casting in CRUD operations now logs warnings instead of silent failures
- Accept header parsing now handles malformed headers gracefully
- Database connection validation in generators prevents confusing errors

### Security
- **Automatic Sensitive Field Protection**: Passwords and tokens automatically excluded from API responses
- **No Breaking Password Hashing**: Existing password authentication remains compatible

---

## Implementation Details

### DTO Architecture

**Request DTOs (Input)**:
```go
type UserCreateDTO struct {
    Firstname string  `json:"firstname"`
    Lastname  string  `json:"lastname"`
    Email     string  `json:"email"`
    Password  *string `json:"password"`
    // id, created_at, updated_at automatically excluded
}
```

**Response DTOs (Output)**:
```go
type UserDTO struct {
    Id        string     `json:"id"`
    Firstname string     `json:"firstname"`
    Lastname  string     `json:"lastname"`
    Email     string     `json:"email"`
    // Password automatically excluded (sensitive field)
    CreatedAt *time.Time `json:"created_at"`
    UpdatedAt *time.Time `json:"updated_at"`
}
```

### Migration Guide

**For API Consumers**:
- No action required for GET requests if you're not relying on sensitive fields
- If you were reading password hashes from responses, this will now fail (security improvement)
- POST/PUT request bodies remain compatible

**For Developers**:
- Run `make generate` to regenerate all code with new DTO support
- Review generated DTOs in `internal/api/dtos/`
- Check generated resources in `internal/api/resources/` for conversion functions

### Breaking Changes Summary

1. **Response Format**: Models → DTOs (sensitive fields removed)
2. **Route Registration**: Auto-generated (no manual intervention needed)
3. **Request Validation**: DTOs enforce field restrictions at type level

---

## Testing

All changes have been tested with:
- ✅ Compilation verification (`go build ./...`)
- ✅ Code generation workflow
- ✅ Database connection validation
- ✅ Dependencies cleanup (`go mod tidy`)

---

## Contributors

- Claude Code (AI Assistant)
- Nicolas Bonnici (@nicolasbonnici)

---

## Notes

This is a release candidate. Please report any issues at:
https://github.com/nicolasbonnici/gorest/issues

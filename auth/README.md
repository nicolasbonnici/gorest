# GoREST Authentication

Built-in JWT-based authentication system for GoREST applications.

## Features

- **JWT Authentication**: Secure token-based authentication with configurable TTL
- **User Management**: Registration, login, and token refresh endpoints
- **Password Security**: Bcrypt password hashing with industry-standard cost
- **Automatic Migrations**: User table created automatically when enabled
- **RBAC Integration**: Seamless integration with GoREST RBAC system
- **Multi-Database**: Compatible with PostgreSQL, MySQL, and SQLite
- **Context Helpers**: Easy access to authenticated user information

## Quick Start

### 1. Enable Auth in Configuration

Add auth configuration to your `gorest.yaml`:

```yaml
auth:
  enabled: true
  jwt_secret: ${JWT_SECRET}
  jwt_ttl: 900  # 15 minutes in seconds
```

Set the JWT secret in your environment:

```bash
export JWT_SECRET="your-super-secret-jwt-key-minimum-32-characters-long"
```

### 2. Auth is Automatically Initialized

No code changes needed! Auth is automatically initialized when `auth.enabled: true` and migrations run automatically.

### 3. Use Auth Endpoints

**Register a new user:**
```bash
POST /auth/register
{
  "email": "user@example.com",
  "password": "securepassword123",
  "firstname": "John",
  "lastname": "Doe"
}
```

**Login:**
```bash
POST /auth/login
{
  "email": "user@example.com",
  "password": "securepassword123"
}
```

**Refresh token:**
```bash
POST /auth/refresh
{
  "token": "your-jwt-token"
}
```

## Protecting Routes

To protect routes, use the auth service's middleware:

```go
package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest"
	"github.com/nicolasbonnici/gorest/auth"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/plugin"
)

func registerRoutes(router fiber.Router, db database.Database, paginationLimit, paginationMaxLimit int, pluginRegistry *plugin.PluginRegistry) {
	// Get auth service (available if auth is enabled)
	// For now, you'll need to initialize it manually in your routes
	// This will be improved in future versions

	// Public routes
	router.Get("/public", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Public endpoint"})
	})

	// Protected routes - use auth middleware from plugin if needed
	// In future versions, auth service will be passed to RegisterRoutes
}
```

## Accessing Authenticated User

Use the auth context helpers to get user information:

```go
import (
	"github.com/gofiber/fiber/v2"
	authpkg "github.com/nicolasbonnici/gorest/auth"
)

func myHandler(c *fiber.Ctx) error {
	// Get authenticated user (returns nil if not authenticated)
	user := authpkg.GetAuthenticatedUser(c)
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "authentication required",
		})
	}

	// Use user ID
	userID := user.UserID

	return c.JSON(fiber.Map{
		"message": "Hello authenticated user",
		"user_id": userID,
	})
}
```

## Database Schema

The auth system creates a `users` table:

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key (auto-generated) |
| `email` | VARCHAR(255) | User email (unique) |
| `password` | TEXT | Bcrypt-hashed password |
| `firstname` | TEXT | User's first name |
| `lastname` | TEXT | User's last name |
| `role` | VARCHAR(50) | User role (default: 'user') |
| `created_at` | TIMESTAMP | Account creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |

**Indexes:**
- Unique index on `email`
- Index on `role` for RBAC queries

## Configuration Reference

```yaml
auth:
  enabled: bool         # Enable/disable authentication (default: false)
  jwt_secret: string    # JWT signing secret (required, min 32 chars)
  jwt_ttl: int          # Token TTL in seconds (default: 900 = 15 min)
```

## Security Best Practices

### JWT Secret

- **Never** commit your JWT secret to version control
- Use a strong, random secret (minimum 32 characters)
- Generate a secure secret:
  ```bash
  openssl rand -base64 32
  ```

### Password Requirements

- Minimum 8 characters enforced
- Bcrypt hashing with default cost (10)
- Stored passwords are never returned in API responses

### HTTPS in Production

Always use HTTPS in production to prevent token interception.

## Error Handling

The auth system returns standard HTTP status codes:

| Status | Meaning |
|--------|---------|
| `201` | User successfully registered |
| `200` | Login successful / Token refreshed |
| `400` | Invalid request body |
| `401` | Invalid credentials / Expired token |
| `409` | User already exists |
| `500` | Internal server error |

## User Model

The User model is defined in `auth/models/user.go`:

```go
type User struct {
    ID        uuid.UUID  `json:"id"`
    Firstname string     `json:"firstname"`
    Lastname  string     `json:"lastname"`
    Email     string     `json:"email"`
    Password  *string    `json:"-"` // Never exposed in JSON
    Role      string     `json:"role"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt *time.Time `json:"updated_at,omitempty"`
}
```

### Password Methods

```go
// Hash password before saving
err := user.HashPassword()

// Verify password during login
isValid := user.CheckPassword("plaintextpassword")
```

## RBAC Integration

Auth automatically integrates with GoREST's RBAC system. The user's role is:

1. Loaded from the database during authentication
2. Stored in the request context
3. Available to RBAC middleware for access control

Configure role-based access in `gorest.yaml`:

```yaml
rbac:
  default_policy: deny_all
  superuser_role: admin
  role_hierarchy:
    admin: [moderator, user]
    moderator: [user]
```

## Migrations

Auth migrations run automatically when:
- `auth.enabled: true` in config
- GoREST migration system is running

The `users` table is created on first startup.

## API Package Usage

You can also use the auth package programmatically:

```go
import (
	"github.com/nicolasbonnici/gorest/auth"
	"github.com/nicolasbonnici/gorest/auth/jwt"
	"github.com/nicolasbonnici/gorest/config"
)

// Initialize auth service
authService, err := auth.NewService(config.AuthConfig{
	Enabled:   true,
	JWTSecret: "your-secret",
	JWTTTL:    900,
}, db)

// Get middleware
authMiddleware := authService.Middleware()
optionalAuthMiddleware := authService.OptionalMiddleware()

// Register routes
authService.RegisterRoutes(router)
```

## Troubleshooting

### "jwt_secret is required"

Ensure `JWT_SECRET` environment variable is set and `auth.jwt_secret` references it in config.

### "jwt_secret must be at least 32 characters"

Use a longer secret for security. Generate one with:
```bash
openssl rand -base64 32
```

### "missing authorization header"

Include the JWT token in the `Authorization` header:
```
Authorization: Bearer <your-token>
```

### "invalid or expired token"

- Token may have expired (default: 15 minutes)
- Use `/auth/refresh` to get a new token
- Verify `JWT_SECRET` matches between token creation and validation

## Related Documentation

- [Main GoREST README](../README.md)
- [RBAC Documentation](../rbac/README.md)
- [Migrations Documentation](../migrations/README.md)
- [Configuration Guide](../CONFIGURATION.md)

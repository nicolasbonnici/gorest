# GoREST Authentication

Built-in JWT-based authentication system for GoREST applications.

## Features

- **JWT Authentication**: Secure token-based authentication with configurable TTL
- **User Management**: Registration, login, logout and token refresh endpoints
- **Refresh Tokens**: Opaque, database-backed tokens with rotation, revocation and reuse detection
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
  jwt_ttl: 900          # access token TTL, 15 minutes in seconds
  refresh_ttl: 2592000  # refresh token TTL, 30 days in seconds
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

Register and login both return an access token, a refresh token and the access
token lifetime in seconds:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "kQ8f3mZ1r...",
  "expires_in": 900,
  "user": { "id": "...", "email": "user@example.com" }
}
```

**Refresh:**
```bash
POST /auth/refresh
{
  "refresh_token": "kQ8f3mZ1r..."
}
```

Returns a new access token *and a new refresh token*. The presented refresh
token is revoked in the same step, so clients must store the returned
`refresh_token` and discard the old one:

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "9dLm2xQ7v...",
  "expires_in": 900
}
```

**Logout:**
```bash
POST /auth/logout
{
  "refresh_token": "9dLm2xQ7v..."
}
```

Responds `204 No Content`. Revoking an unknown or already-revoked token is not
an error, so logout is safe to retry. Logout revokes the refresh token only —
any access token already issued stays valid until it expires, which is why
`jwt_ttl` should stay short.

## Refresh Token Model

Access tokens are stateless JWTs; refresh tokens are opaque random strings
tracked in the database. Only a SHA-256 hash of each token is stored, so the
table is not a credential store even if it leaks.

Every refresh **rotates**: the old token is revoked and chained to its
replacement via `replaced_by`. If a revoked token is ever presented again, that
implies it leaked, and the server revokes the user's entire token family and
returns `401`. The client must then log in again.

Because rotation makes reuse detectable, a stolen refresh token gives an
attacker a session only until the legitimate client next refreshes.

## Protecting Routes

To protect routes, use the auth service's middleware:

```go
package main

import (
	"github.com/gofiber/fiber/v3"
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
	router.Get("/public", func(c fiber.Ctx) error {
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
	"github.com/gofiber/fiber/v3"
	authpkg "github.com/nicolasbonnici/gorest/auth"
)

func myHandler(c fiber.Ctx) error {
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
| `created_at` | TIMESTAMP | Account creation timestamp |
| `updated_at` | TIMESTAMP | Last update timestamp |

**Indexes:** unique index on `email`.

Roles are not a column on `users` — they live in `user_roles`/`roles` and are
loaded per request. See [RBAC Integration](#rbac-integration).

And a `refresh_tokens` table:

| Column | Type | Description |
|--------|------|-------------|
| `id` | UUID | Primary key |
| `user_id` | UUID | Owning user (FK, cascades on delete) |
| `token_hash` | TEXT | SHA-256 of the token (unique); the plaintext is never stored |
| `expires_at` | BIGINT | Expiry as epoch seconds |
| `revoked_at` | BIGINT | Revocation time as epoch seconds, `NULL` while active |
| `replaced_by` | UUID | The token this one was rotated into |
| `created_at` | TIMESTAMP | Issue timestamp |

**Indexes:** unique index on `token_hash`, index on `user_id`.

Expiry is stored as epoch seconds rather than a native timestamp so comparisons
and scans behave identically on PostgreSQL, MySQL and SQLite.

## Configuration Reference

```yaml
auth:
  enabled: bool         # Enable/disable authentication (default: false)
  jwt_secret: string    # JWT signing secret (required, min 32 chars)
  jwt_ttl: int          # Access token TTL in seconds (default: 900 = 15 min)
  refresh_ttl: int      # Refresh token TTL in seconds (default: 2592000 = 30 days)
```

`refresh_ttl` must be greater than `jwt_ttl`; startup fails otherwise, since a
refresh token that expires before the access token it renews is useless.

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

### Refresh Tokens

- Store the refresh token where it is least exposed to scripts on the client;
  it is long-lived and grants new sessions
- Always persist the `refresh_token` returned by `/auth/refresh` — the previous
  one is dead the moment it is used
- Keep `jwt_ttl` short. Revocation acts on refresh tokens, so an already-issued
  access token survives logout until it expires
- Call `/auth/logout` on sign-out so the refresh token cannot be replayed

### HTTPS in Production

Always use HTTPS in production to prevent token interception.

## Error Handling

The auth system returns standard HTTP status codes:

| Status | Meaning |
|--------|---------|
| `201` | User successfully registered |
| `200` | Login successful / Token refreshed |
| `204` | Logout successful |
| `400` | Invalid request body |
| `401` | Invalid credentials / Expired token / Invalid or reused refresh token |
| `409` | User already exists |
| `500` | Internal server error |

A refresh rejected for reuse returns `401` with
`refresh token reuse detected, all sessions revoked`, distinguishing it from an
ordinary expiry so clients can stop retrying and force a fresh login.

## User Model

The User model is defined in `auth/models/user.go`:

```go
type User struct {
    ID        uuid.UUID  `json:"id"`
    Firstname string     `json:"firstname"`
    Lastname  string     `json:"lastname"`
    Email     string     `json:"email"`
    Password  *string    `json:"-"` // Never exposed in JSON
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
	Enabled:    true,
	JWTSecret:  "your-secret",
	JWTTTL:     900,
	RefreshTTL: 2592000,
}, db)

// Get middleware
authMiddleware := authService.Middleware()
optionalAuthMiddleware := authService.OptionalMiddleware()

// Register routes
authService.RegisterRoutes(router)
```

### Managing Refresh Tokens

`RefreshService()` exposes the token store for cases the endpoints don't cover,
such as ending every session after a password change:

```go
err := authService.RefreshService().RevokeAllForUser(ctx, userID)
```

Expired rows are swept hourly by a goroutine that `gorest.Start` launches and
cancels on shutdown. If you build your own server, either start it yourself or
run the sweep on your own schedule:

```go
authService.StartRefreshTokenCleanup(ctx)

// ...or manually
deleted, err := authService.RefreshService().DeleteExpired(ctx, time.Now())
```

Revoked tokens are deliberately kept until they expire — reuse detection needs
them to recognise a replayed token.

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

### "invalid or expired refresh token"

- The refresh token expired (default: 30 days), or was already rotated away by
  an earlier `/auth/refresh` call. Check the client stores the new
  `refresh_token` from every refresh response
- The user must log in again

### "refresh token reuse detected, all sessions revoked"

A refresh token was presented after it had already been used. Every session for
that user is now revoked and the user must log in again.

Usually this is a client bug — two requests refreshing concurrently, or a retry
replaying the old token — rather than a real theft. Serialize refreshes on the
client so only one is in flight at a time.

### "auth.refresh_ttl must be greater than auth.jwt_ttl"

The refresh token would expire before the access token it exists to renew.
Raise `refresh_ttl` or lower `jwt_ttl`.

## Related Documentation

- [Main GoREST README](../README.md)
- [RBAC Documentation](../RBAC.md)
- [Migrations Documentation](../migrations/README.md)
- [Configuration Guide](../CONFIGURATION.md)

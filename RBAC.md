# Role-Based Access Control (RBAC) in GoREST

GoREST provides a powerful, flexible RBAC system as a mandatory 5th hook layer. This guide covers everything you need to implement authorization in your API.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Struct Tag Syntax](#struct-tag-syntax)
- [Authorization Hook Methods](#authorization-hook-methods)
- [Context Management](#context-management)
- [Role Hierarchy](#role-hierarchy)
- [Common Patterns](#common-patterns)
- [Security Considerations](#security-considerations)
- [Troubleshooting](#troubleshooting)

## Overview

The RBAC system provides:

- **Field-level permissions** via struct tags (`rbac:"read:roles;write:roles"`)
- **Resource-level permissions** via Authorization hook methods
- **Role hierarchy** with inheritance
- **Automatic field filtering** for unauthorized reads
- **Flexible policies** (deny-by-default or allow-by-default)

### Architecture

```
Request → Middleware → CRUD Operation
                         ↓
                    Authorization Layer (5th Hook)
                         ↓
                    ┌─────────────────────┐
                    │ ValidateWrite()     │ ← Check field permissions (Create/Update)
                    │ CheckCreate()       │ ← Resource-level check (Create)
                    │ CheckRead()         │ ← Resource-level check (Read)
                    │ CheckUpdate()       │ ← Resource-level check (Update)
                    │ CheckDelete()       │ ← Resource-level check (Delete)
                    │ FilterRead()        │ ← Remove forbidden fields (Read)
                    └─────────────────────┘
                         ↓
                    StateProcessor (1st Hook)
                         ↓
                    Database
```

## Quick Start

### 1. Add RBAC Tags to Your Model

```go
type Article struct {
    ID        string    `json:"id" db:"id" rbac:"read:*;write:*"`
    Title     string    `json:"title" db:"title" rbac:"read:*;write:editor,admin"`
    Content   string    `json:"content" db:"content" rbac:"read:*;write:editor,admin"`
    Published bool      `json:"published" db:"published" rbac:"read:*;write:admin"`
    AuthorID  string    `json:"author_id" db:"author_id" rbac:"read:*;write:none"`
    CreatedAt time.Time `json:"created_at" db:"created_at" rbac:"read:*;write:none"`
}

func (Article) TableName() string { return "articles" }
```

### 2. Create Hooks with Default Authorization

```go
type ArticleHooks struct {
    *hooks.DefaultAuthorization[models.Article]
    hooks.NoOpHooks[models.Article]
}

func NewArticleHooks() *ArticleHooks {
    return &ArticleHooks{
        DefaultAuthorization: hooks.NewDefaultAuthorization[models.Article](
            hooks.GetGlobalRBACConfig(),
        ),
    }
}
```

### 3. Configure RBAC in gorest.yaml

```yaml
rbac:
  default_policy: deny_all
  superuser_role: admin
  default_field_policy: deny
  strict_validation: false
  cache_enabled: true
  cache_ttl: 300

  role_hierarchy:
    admin:
      - editor
      - user
    editor:
      - user
```

### 4. Set Roles in Middleware

```go
// In your auth middleware
func AuthMiddleware(c *fiber.Ctx) error {
    // Extract user from JWT/session
    user := getUserFromToken(c)

    // Set RBAC context
    ctx := c.Context()
    ctx = rbac.WithRoles(ctx, user.Roles)
    ctx = rbac.WithUserID(ctx, user.ID)

    c.SetUserContext(ctx)
    return c.Next()
}
```

That's it! Your API now has field-level and resource-level authorization.

## Configuration

### YAML Configuration

```yaml
rbac:
  # Policy when no explicit permission exists
  # Options: "deny_all" (recommended), "allow_all"
  default_policy: deny_all

  # Role that bypasses all permission checks
  superuser_role: admin

  # Policy for fields without rbac tags
  # Options: "deny" (recommended), "allow"
  default_field_policy: deny

  # Validate zero-value fields during writes
  # true = more secure (validates all fields)
  # false = backwards compatible (skips zero values)
  strict_validation: false

  # Cache permission parsing results
  cache_enabled: true
  cache_ttl: 300  # seconds

  # Role inheritance (parent inherits child permissions)
  role_hierarchy:
    admin:
      - moderator
      - editor
      - user
    moderator:
      - user
    editor:
      - user
```

### Programmatic Configuration

```go
import "github.com/nicolasbonnici/gorest/rbac"

config := rbac.Config{
    DefaultPolicy:      rbac.DenyAll,
    SuperuserRole:      "admin",
    RoleHierarchy:      map[string][]string{
        "admin": {"editor", "user"},
        "editor": {"user"},
    },
    DefaultFieldPolicy: "deny",
    StrictValidation:   false,
    CacheEnabled:       true,
    CacheTTL:           300,
}

// Set global config (used by DefaultAuthorization)
hooks.SetGlobalRBACConfig(config)
```

## Struct Tag Syntax

### Basic Syntax

```go
`rbac:"read:roles;write:roles"`
```

### Special Role Values

- `*` - Public access (everyone, including unauthenticated)
- `any` - Any authenticated user
- `none` - No one (field cannot be read/written)
- `role1,role2` - Specific roles (comma-separated)

### Examples

```go
type User struct {
    // Public read, any authenticated user can write
    ID string `rbac:"read:*;write:any"`

    // Public read, only user can write (ownership check in hook)
    Name string `rbac:"read:*;write:user"`

    // Only admin can read/write
    Email string `rbac:"read:admin;write:admin"`

    // Public read, only admin and moderator can write
    Status string `rbac:"read:*;write:admin,moderator"`

    // Anyone can read, no one can write (managed by system)
    CreatedAt time.Time `rbac:"read:*;write:none"`

    // Only specific roles can read
    InternalNotes string `rbac:"read:admin,moderator;write:admin"`
}
```

### Read-Only vs Write-Only

```go
// Read-only field
Password string `rbac:"read:none;write:user"`

// Write-only field (uncommon)
Token string `rbac:"read:admin;write:*"`

// System-managed (no one can write)
UpdatedAt time.Time `rbac:"read:*;write:none"`
```

## Authorization Hook Methods

### Default Behavior

`DefaultAuthorization[T]` provides tag-based authorization out of the box:

- `CheckCreate()` - Allows if user has any role (deny_all) or always (allow_all)
- `CheckRead()` - Allows if user has any role (deny_all) or always (allow_all)
- `CheckUpdate()` - Allows if user has any role (deny_all) or always (allow_all)
- `CheckDelete()` - Allows if user has any role (deny_all) or always (allow_all)
- `ValidateWrite()` - Checks field-level write permissions via tags
- `FilterRead()` - Removes fields user cannot read via tags

### Custom Resource-Level Checks

Override methods for complex business logic:

#### Example: Ownership Check

```go
type TodoHooks struct {
    *hooks.DefaultAuthorization[models.Todo]
    hooks.NoOpHooks[models.Todo]
}

func (h *TodoHooks) CheckRead(ctx context.Context, todo *models.Todo) error {
    userID, ok := rbac.GetUserID(ctx)
    if !ok {
        return rbac.ErrPermissionDenied
    }

    roles, _ := rbac.GetRoles(ctx)

    // Admins can read all todos
    if h.GetVoter().IsSuperuser(roles) {
        return nil
    }

    // Users can only read their own todos
    if todo.UserID != nil && *todo.UserID != userID {
        return rbac.ErrNotFound  // 404 for security (don't disclose existence)
    }

    return nil
}

func (h *TodoHooks) CheckUpdate(ctx context.Context, id any, todo *models.Todo) error {
    // Similar ownership check for updates
    // ...
}
```

#### Example: Status-Based Access

```go
func (h *ArticleHooks) CheckRead(ctx context.Context, article *models.Article) error {
    roles, _ := rbac.GetRoles(ctx)

    // Published articles are public
    if article.Published {
        return nil
    }

    // Unpublished articles only visible to authors and editors
    userID, _ := rbac.GetUserID(ctx)
    if article.AuthorID == userID {
        return nil
    }

    if rbac.HasAnyRole(roles, []string{"editor", "admin"}, h.GetVoter().GetConfig().RoleHierarchy) {
        return nil
    }

    return rbac.ErrNotFound
}
```

#### Example: Custom Validation

```go
func (h *UserHooks) ValidateWrite(ctx context.Context, user *models.User) error {
    // First check field-level permissions
    if err := h.DefaultAuthorization.ValidateWrite(ctx, user); err != nil {
        return err
    }

    // Custom validation: users can't change their own role
    userID, _ := rbac.GetUserID(ctx)
    if user.ID == userID && user.Role != "" {
        return fmt.Errorf("users cannot change their own role")
    }

    return nil
}
```

### Method Signatures

```go
type Authorization[T any] interface {
    // Check if user can create this resource
    CheckCreate(ctx context.Context, model *T) error

    // Check if user can read this resource (return ErrNotFound for 404)
    CheckRead(ctx context.Context, model *T) error

    // Check if user can update this resource
    CheckUpdate(ctx context.Context, id any, model *T) error

    // Check if user can delete this resource
    CheckDelete(ctx context.Context, id any) error

    // Filter fields user cannot read (modifies model in place)
    FilterRead(ctx context.Context, model *T) error

    // Validate user can write all non-zero fields
    ValidateWrite(ctx context.Context, model *T) error

    // Get underlying voter for role checks
    GetVoter() rbac.Voter
}
```

## Context Management

### Setting Roles

```go
import "github.com/nicolasbonnici/gorest/rbac"

// Set roles only
ctx = rbac.WithRoles(ctx, []string{"user", "editor"})

// Set user ID only
ctx = rbac.WithUserID(ctx, "user-123")

// Set both
ctx = rbac.WithUser(ctx, "user-123", []string{"user", "editor"})
```

### Getting Roles

```go
// Get roles
roles, ok := rbac.GetRoles(ctx)
if !ok {
    // No roles set (unauthenticated)
}

// Get user ID
userID, ok := rbac.GetUserID(ctx)
if !ok {
    // No user ID set
}
```

### Typical Middleware Pattern

```go
func AuthMiddleware(c *fiber.Ctx) error {
    token := c.Get("Authorization")

    if token == "" {
        // Unauthenticated - no roles set
        return c.Next()
    }

    claims, err := ValidateJWT(token)
    if err != nil {
        return c.Status(401).JSON(fiber.Map{"error": "Invalid token"})
    }

    // Set RBAC context
    ctx := c.Context()
    ctx = rbac.WithUser(ctx, claims.UserID, claims.Roles)
    c.SetUserContext(ctx)

    return c.Next()
}
```

## Role Hierarchy

### How It Works

Role hierarchy allows roles to inherit permissions from other roles:

```yaml
role_hierarchy:
  admin:
    - moderator
    - user
  moderator:
    - user
```

With this hierarchy:
- `admin` has permissions of `admin`, `moderator`, and `user`
- `moderator` has permissions of `moderator` and `user`
- `user` has only `user` permissions

### Checking Roles with Hierarchy

```go
import "github.com/nicolasbonnici/gorest/rbac"

roles := []string{"moderator"}
hierarchy := map[string][]string{
    "admin": {"moderator", "user"},
    "moderator": {"user"},
}

// Check for specific role
if rbac.HasRole(roles, "user", hierarchy) {
    // true - moderator inherits user
}

// Check for any role
if rbac.HasAnyRole(roles, []string{"admin", "user"}, hierarchy) {
    // true - moderator inherits user
}

// Resolve all roles
expanded := rbac.ResolveRoles(roles, hierarchy)
// []string{"moderator", "user"}
```

### Circular Dependency Protection

The system validates role hierarchies on startup:

```go
// This would fail validation
role_hierarchy:
  admin:
    - moderator
  moderator:
    - admin  # ERROR: circular dependency
```

## Common Patterns

### Pattern 1: Public Read, Restricted Write

```go
type BlogPost struct {
    ID      string `rbac:"read:*;write:*"`
    Title   string `rbac:"read:*;write:author,admin"`
    Content string `rbac:"read:*;write:author,admin"`
    Author  string `rbac:"read:*;write:none"`
}
```

### Pattern 2: Ownership-Based Access

```go
type Document struct {
    ID      string  `rbac:"read:any;write:any"`
    Title   string  `rbac:"read:any;write:any"`
    OwnerID *string `rbac:"read:any;write:none"`
}

func (h *DocumentHooks) CheckRead(ctx context.Context, doc *Document) error {
    userID, _ := rbac.GetUserID(ctx)
    roles, _ := rbac.GetRoles(ctx)

    // Admins see all
    if h.GetVoter().IsSuperuser(roles) {
        return nil
    }

    // Owners see their own
    if doc.OwnerID != nil && *doc.OwnerID == userID {
        return nil
    }

    return rbac.ErrNotFound
}
```

### Pattern 3: Progressive Disclosure

```go
type User struct {
    ID       string `rbac:"read:*;write:*"`
    Name     string `rbac:"read:*;write:user,admin"`
    Email    string `rbac:"read:user,admin;write:user,admin"`
    Phone    string `rbac:"read:admin;write:admin"`
    Password string `rbac:"read:none;write:user"`
}
```

### Pattern 4: Soft Delete

```go
type Record struct {
    ID        string     `rbac:"read:*;write:*"`
    DeletedAt *time.Time `rbac:"read:admin;write:none"`
}

func (h *RecordHooks) ModifySelectQuery(ctx context.Context, op hooks.Operation, builder *query.SelectBuilder) (*query.SelectBuilder, bool) {
    roles, _ := rbac.GetRoles(ctx)

    // Non-admins don't see soft-deleted records
    if !h.GetVoter().IsSuperuser(roles) {
        builder = builder.Where(query.Eq("deleted_at", nil))
        return builder, true
    }

    return builder, false
}

func (h *RecordHooks) CheckDelete(ctx context.Context, id any) error {
    // Soft delete - mark as deleted instead of removing
    // Actual deletion in StateProcessor
    return nil
}
```

### Pattern 5: Multi-Tenant Isolation

```go
type Product struct {
    ID       string `rbac:"read:any;write:any"`
    Name     string `rbac:"read:any;write:any"`
    TenantID string `rbac:"read:any;write:none"`
}

func (h *ProductHooks) ModifySelectQuery(ctx context.Context, op hooks.Operation, builder *query.SelectBuilder) (*query.SelectBuilder, bool) {
    tenantID, ok := ctx.Value("tenant_id").(string)
    if !ok {
        return builder, false
    }

    roles, _ := rbac.GetRoles(ctx)

    // System admins see all tenants
    if h.GetVoter().IsSuperuser(roles) {
        return builder, false
    }

    // Regular users see only their tenant
    builder = builder.Where(query.Eq("tenant_id", tenantID))
    return builder, true
}
```

## Security Considerations

### 1. Always Return 404, Not 403 for Reads

```go
// Good - hides existence
if err := h.CheckRead(ctx, resource); err != nil {
    return rbac.ErrNotFound
}

// Bad - discloses existence
if err := h.CheckRead(ctx, resource); err != nil {
    return rbac.ErrPermissionDenied
}
```

### 2. Validate Writes Before State Changes

The system automatically validates writes before StateProcessor, but custom logic should also validate early:

```go
func (h *MyHooks) StateProcessor(ctx context.Context, op hooks.Operation, id any, model *MyModel) error {
    // ValidateWrite already called before this
    // Focus on business logic, not permissions
    return nil
}
```

### 3. Use Strict Validation for Sensitive Data

```yaml
rbac:
  strict_validation: true  # Validates even zero values
```

This prevents attackers from bypassing validation by setting restricted fields to zero values.

### 4. Superuser Role is Powerful

The superuser role bypasses ALL checks. Use carefully:

```yaml
rbac:
  superuser_role: admin  # Only give to truly trusted users
```

### 5. Default Deny is Safest

```yaml
rbac:
  default_policy: deny_all           # Recommended
  default_field_policy: deny         # Recommended
```

### 6. Audit Sensitive Operations

```go
func (h *UserHooks) CheckDelete(ctx context.Context, id any) error {
    userID, _ := rbac.GetUserID(ctx)
    log.Warn("User deletion", "deleted_id", id, "actor", userID)
    return nil
}
```

## Troubleshooting

### "permission denied" on Create/Update

**Cause**: Field-level write permission check failed

**Solution**:
1. Check rbac tags on model fields
2. Verify user has required roles in context
3. Check if field has zero value (set `strict_validation: true`)

```go
// Add debug logging
func (h *MyHooks) ValidateWrite(ctx context.Context, model *MyModel) error {
    roles, _ := rbac.GetRoles(ctx)
    log.Debug("ValidateWrite", "roles", roles)
    return h.DefaultAuthorization.ValidateWrite(ctx, model)
}
```

### "forbidden fields" error

**Cause**: User trying to write fields they don't have permission for

**Solution**:
1. Update rbac tags to allow write: `rbac:"read:*;write:user,admin"`
2. Or remove the field from the write operation
3. Or ensure user has the required role

### CheckRead returns 404 for everything

**Cause**: Custom CheckRead logic is too restrictive

**Solution**:
```go
func (h *MyHooks) CheckRead(ctx context.Context, model *MyModel) error {
    // Call parent first for tag-based checks
    if err := h.DefaultAuthorization.CheckRead(ctx, model); err != nil {
        return err
    }

    // Then add custom logic
    // ...
}
```

### Fields are missing in response

**Cause**: FilterRead is removing fields based on rbac tags

**Solution**: Check field tags:
```go
// This field will be missing for non-admins
Secret string `rbac:"read:admin;write:admin"`

// Make it public
Secret string `rbac:"read:*;write:admin"`
```

### Role hierarchy not working

**Cause**: Role hierarchy not configured or has circular dependency

**Solution**:
1. Check `gorest.yaml` has correct hierarchy
2. Verify no circular dependencies
3. Check logs for validation errors on startup

```yaml
role_hierarchy:
  admin:
    - editor
    - user
  editor:
    - user
```

### Performance issues

**Cause**: Permission parsing happening on every request

**Solution**: Enable caching
```yaml
rbac:
  cache_enabled: true
  cache_ttl: 300
```

---

**See Also:**
- [AUTHORIZATION.md](AUTHORIZATION.md) - Authorization hook layer reference
- [HOOKS.md](HOOKS.md) - Complete hook system documentation
- [CONFIGURATION.md](CONFIGURATION.md) - RBAC configuration reference

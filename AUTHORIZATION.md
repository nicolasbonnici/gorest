# Authorization Hook Layer Reference

The Authorization layer is the 5th hook layer in GoREST, providing mandatory role-based access control (RBAC) for all resources.

## Table of Contents

- [Overview](#overview)
- [Interface Definition](#interface-definition)
- [DefaultAuthorization Implementation](#defaultauthorization-implementation)
- [Method Reference](#method-reference)
- [Integration with CRUD](#integration-with-crud)
- [Custom Implementations](#custom-implementations)
- [Error Handling](#error-handling)

## Overview

The Authorization layer is **mandatory** - all resources must implement the `Authorization[T]` interface. This can be done by:

1. **Embedding `DefaultAuthorization[T]`** - Provides tag-based permissions (most common)
2. **Implementing custom methods** - Override specific methods for complex logic
3. **Full custom implementation** - Implement all methods from scratch (rare)

### Hook Layer Order

```
HTTP Request
  ↓
Layer 5: Authorization ← YOU ARE HERE
  ↓
Layer 1: StateProcessor
  ↓
Layer 3: SQLQueryBuilderModifier
  ↓
Layer 2: SQLQueryListener (BeforeQuery)
  ↓
Database Execution
  ↓
Layer 2: SQLQueryListener (AfterQuery)
  ↓
Layer 4: Serializer
  ↓
HTTP Response
```

## Interface Definition

```go
package hooks

type Authorization[T any] interface {
    // Check if user can create this resource
    // Called before ValidateWrite and StateProcessor
    // Return error to deny (403 Forbidden)
    CheckCreate(ctx context.Context, model *T) error

    // Check if user can read this resource
    // Called after database query, before FilterRead
    // Return error to deny (404 Not Found for security)
    CheckRead(ctx context.Context, model *T) error

    // Check if user can update this resource
    // Called before ValidateWrite and StateProcessor
    // Return error to deny (403 Forbidden)
    CheckUpdate(ctx context.Context, id any, model *T) error

    // Check if user can delete this resource
    // Called before StateProcessor
    // Return error to deny (403 Forbidden)
    CheckDelete(ctx context.Context, id any) error

    // Filter fields user cannot read
    // Called after CheckRead, modifies model in place
    // Return error for system errors only
    FilterRead(ctx context.Context, model *T) error

    // Validate user can write all non-zero fields
    // Called before CheckCreate/CheckUpdate
    // Return error to deny (403 Forbidden)
    ValidateWrite(ctx context.Context, model *T) error

    // Get underlying voter for advanced role checks
    GetVoter() rbac.Voter
}
```

## DefaultAuthorization Implementation

### Basic Usage

```go
import (
    "github.com/nicolasbonnici/gorest/hooks"
    "github.com/nicolasbonnici/gorest/rbac"
)

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

### What DefaultAuthorization Provides

| Method | Behavior |
|--------|----------|
| `CheckCreate()` | Allows if user has any role (deny_all policy) or always (allow_all policy) |
| `CheckRead()` | Allows if user has any role (deny_all policy) or always (allow_all policy) |
| `CheckUpdate()` | Allows if user has any role (deny_all policy) or always (allow_all policy) |
| `CheckDelete()` | Allows if user has any role (deny_all policy) or always (allow_all policy) |
| `ValidateWrite()` | Checks field-level write permissions via `rbac:` tags |
| `FilterRead()` | Removes fields user cannot read via `rbac:` tags |
| `GetVoter()` | Returns configured voter instance |

### Superuser Bypass

All `DefaultAuthorization` methods check for the superuser role first:

```go
roles, _ := rbac.GetRoles(ctx)
if h.GetVoter().IsSuperuser(roles) {
    return nil  // Bypass all checks
}
```

## Method Reference

### CheckCreate

**When Called**: Before Create operation, after ValidateWrite

**Purpose**: Check if user can create this type of resource

**Return**:
- `nil` - Allow creation
- `error` - Deny creation (returns 403 Forbidden)

**Default Behavior**: Allows if user has any role (deny_all) or always (allow_all)

**Example Override**:
```go
func (h *ArticleHooks) CheckCreate(ctx context.Context, article *models.Article) error {
    roles, _ := rbac.GetRoles(ctx)

    // Only editors and admins can create articles
    if !rbac.HasAnyRole(roles, []string{"editor", "admin"}, h.GetVoter().GetConfig().RoleHierarchy) {
        return rbac.ErrPermissionDenied
    }

    return nil
}
```

### CheckRead

**When Called**: After database query, before FilterRead

**Purpose**: Check if user can read this specific resource

**Return**:
- `nil` - Allow read
- `error` - Deny read (returns 404 Not Found)

**Important**: Return 404, not 403, to avoid information disclosure

**Default Behavior**: Allows if user has any role (deny_all) or always (allow_all)

**Example Override**:
```go
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
        return rbac.ErrNotFound  // 404 for security
    }

    return nil
}
```

### CheckUpdate

**When Called**: Before Update operation, after ValidateWrite

**Purpose**: Check if user can update this specific resource

**Parameters**:
- `ctx` - Request context with roles
- `id` - Resource ID being updated
- `model` - New data to be written

**Return**:
- `nil` - Allow update
- `error` - Deny update (returns 403 Forbidden)

**Default Behavior**: Allows if user has any role (deny_all) or always (allow_all)

**Example Override**:
```go
func (h *PostHooks) CheckUpdate(ctx context.Context, id any, post *models.Post) error {
    userID, _ := rbac.GetUserID(ctx)
    roles, _ := rbac.GetRoles(ctx)

    // Admins can update all posts
    if h.GetVoter().IsSuperuser(roles) {
        return nil
    }

    // Users can only update their own posts
    // Note: Need to fetch existing post to check ownership
    existing, err := fetchPostByID(ctx, id)
    if err != nil {
        return err
    }

    if existing.AuthorID != userID {
        return rbac.ErrPermissionDenied
    }

    return nil
}
```

### CheckDelete

**When Called**: Before Delete operation, before StateProcessor

**Purpose**: Check if user can delete this specific resource

**Parameters**:
- `ctx` - Request context with roles
- `id` - Resource ID being deleted

**Return**:
- `nil` - Allow deletion
- `error` - Deny deletion (returns 403 Forbidden)

**Default Behavior**: Allows if user has any role (deny_all) or always (allow_all)

**Example Override**:
```go
func (h *CommentHooks) CheckDelete(ctx context.Context, id any) error {
    userID, _ := rbac.GetUserID(ctx)
    roles, _ := rbac.GetRoles(ctx)

    // Admins and moderators can delete any comment
    if rbac.HasAnyRole(roles, []string{"admin", "moderator"}, h.GetVoter().GetConfig().RoleHierarchy) {
        return nil
    }

    // Users can delete their own comments
    comment, err := fetchCommentByID(ctx, id)
    if err != nil {
        return err
    }

    if comment.AuthorID != userID {
        return rbac.ErrPermissionDenied
    }

    return nil
}
```

### FilterRead

**When Called**: After CheckRead passes, before Serializer

**Purpose**: Remove fields user cannot read from the model

**Parameters**:
- `ctx` - Request context with roles
- `model` - Resource to filter (modified in place)

**Return**:
- `nil` - Filtering succeeded
- `error` - System error (not permission denied)

**Default Behavior**: Uses `rbac:` tags to set forbidden fields to zero values

**Example Override**:
```go
func (h *UserHooks) FilterRead(ctx context.Context, user *models.User) error {
    // First apply tag-based filtering
    if err := h.DefaultAuthorization.FilterRead(ctx, user); err != nil {
        return err
    }

    // Additional custom filtering
    userID, _ := rbac.GetUserID(ctx)
    roles, _ := rbac.GetRoles(ctx)

    // Hide email from non-owners (unless admin)
    if user.ID != userID && !h.GetVoter().IsSuperuser(roles) {
        user.Email = ""
    }

    return nil
}
```

### ValidateWrite

**When Called**: Before Create/Update operations, before CheckCreate/CheckUpdate

**Purpose**: Validate user has permission to write all non-zero fields

**Parameters**:
- `ctx` - Request context with roles
- `model` - Resource with fields to write

**Return**:
- `nil` - Validation passed
- `error` - Validation failed (returns 403 Forbidden)

**Default Behavior**:
- Uses `rbac:` tags to check field-level write permissions
- Skips zero-value fields unless `strict_validation: true`
- Returns `ValidationError` with list of forbidden fields

**Example Override**:
```go
func (h *SettingsHooks) ValidateWrite(ctx context.Context, settings *models.Settings) error {
    // First apply tag-based validation
    if err := h.DefaultAuthorization.ValidateWrite(ctx, settings); err != nil {
        return err
    }

    // Additional custom validation
    userID, _ := rbac.GetUserID(ctx)

    // Users can't change settings for other users
    if settings.UserID != userID {
        return fmt.Errorf("cannot modify another user's settings")
    }

    return nil
}
```

### GetVoter

**When Called**: By other methods or your custom code

**Purpose**: Access the underlying voter for role checks

**Return**: `rbac.Voter` instance

**Example Usage**:
```go
func (h *MyHooks) CheckRead(ctx context.Context, model *MyModel) error {
    roles, _ := rbac.GetRoles(ctx)

    // Check if user is superuser
    if h.GetVoter().IsSuperuser(roles) {
        return nil
    }

    // Check specific field permission
    if err := h.GetVoter().CheckRead(ctx, model, "SecretField"); err != nil {
        model.SecretField = ""
    }

    return nil
}
```

## Integration with CRUD

### Create Operation Flow

```go
func (c *CRUD[T]) Create(ctx context.Context, m T) error {
    // 1. ValidateWrite - Check field-level permissions
    if err := c.Hooks.ValidateWrite(ctx, &m); err != nil {
        return fmt.Errorf("authorization failed: %w", err)
    }

    // 2. CheckCreate - Check resource-level permission
    if err := c.Hooks.CheckCreate(ctx, &m); err != nil {
        return fmt.Errorf("authorization failed: %w", err)
    }

    // 3. StateProcessor - Business logic validation
    if err := c.Hooks.StateProcessor(ctx, hooks.OperationCreate, nil, &m); err != nil {
        return err
    }

    // 4. Execute INSERT
    // ...
}
```

### Read Operation Flow

```go
func (c *CRUD[T]) GetByID(ctx context.Context, id any) (*T, error) {
    // 1. Execute SELECT
    // ...

    // 2. CheckRead - Check resource-level permission
    if err := c.Hooks.CheckRead(ctx, &item); err != nil {
        return nil, sql.ErrNoRows  // 404, not 403
    }

    // 3. FilterRead - Remove forbidden fields
    if err := c.Hooks.FilterRead(ctx, &item); err != nil {
        return nil, fmt.Errorf("authorization failed: %w", err)
    }

    // 4. SerializeOne - Format response
    // ...

    return &item, nil
}
```

### Update Operation Flow

```go
func (c *CRUD[T]) Update(ctx context.Context, id any, m T) error {
    // 1. ValidateWrite - Check field-level permissions
    if err := c.Hooks.ValidateWrite(ctx, &m); err != nil {
        return fmt.Errorf("authorization failed: %w", err)
    }

    // 2. CheckUpdate - Check resource-level permission
    if err := c.Hooks.CheckUpdate(ctx, id, &m); err != nil {
        return fmt.Errorf("authorization failed: %w", err)
    }

    // 3. StateProcessor - Business logic validation
    // ...

    // 4. Execute UPDATE
    // ...
}
```

### Delete Operation Flow

```go
func (c *CRUD[T]) Delete(ctx context.Context, id any) error {
    // 1. CheckDelete - Check resource-level permission
    if err := c.Hooks.CheckDelete(ctx, id); err != nil {
        return fmt.Errorf("authorization failed: %w", err)
    }

    // 2. StateProcessor - Business logic validation
    // ...

    // 3. Execute DELETE
    // ...
}
```

## Custom Implementations

### Full Custom Implementation

```go
type CustomAuthorization struct {
    config rbac.Config
    voter  rbac.Voter
}

func (a *CustomAuthorization) CheckCreate(ctx context.Context, model *MyModel) error {
    // Fully custom logic
    return nil
}

// Implement all other methods...
```

### Hybrid Approach (Recommended)

```go
type MyHooks struct {
    *hooks.DefaultAuthorization[models.MyModel]
    hooks.NoOpHooks[models.MyModel]
}

// Override only what you need
func (h *MyHooks) CheckRead(ctx context.Context, model *models.MyModel) error {
    // Custom logic
}

// Other methods use DefaultAuthorization
```

### Composition Pattern

```go
type MyHooks struct {
    *hooks.DefaultAuthorization[models.MyModel]
    hooks.NoOpHooks[models.MyModel]
    ownershipChecker *OwnershipChecker
}

func (h *MyHooks) CheckRead(ctx context.Context, model *models.MyModel) error {
    // Delegate to ownership checker
    return h.ownershipChecker.Check(ctx, model.OwnerID)
}
```

## Error Handling

### Standard Errors

```go
import "github.com/nicolasbonnici/gorest/rbac"

// Permission denied (returns 403)
return rbac.ErrPermissionDenied

// Resource not found (returns 404)
return rbac.ErrNotFound

// Field permission error (returns 403)
return &rbac.FieldPermissionError{
    Field:     "Email",
    Operation: "write",
    Required:  []string{"admin"},
    UserRoles: []string{"user"},
}

// Validation error (returns 403)
return &rbac.ValidationError{
    ForbiddenFields: []string{"Email", "Phone"},
}
```

### HTTP Status Codes

| Operation | Success | Auth Error |
|-----------|---------|------------|
| Create | 201 Created | 403 Forbidden |
| GetAll | 200 OK | Items filtered out |
| GetByID | 200 OK | 404 Not Found |
| Update | 200 OK | 403 Forbidden |
| Delete | 204 No Content | 403 Forbidden |

**Why 404 for reads?** To avoid information disclosure. A 403 reveals the resource exists.

### Error Wrapping

```go
func (h *MyHooks) CheckRead(ctx context.Context, model *MyModel) error {
    if err := validateOwnership(ctx, model); err != nil {
        return fmt.Errorf("ownership check failed: %w", err)
    }
    return nil
}
```

---

**See Also:**
- [RBAC.md](RBAC.md) - Complete RBAC guide
- [HOOKS.md](HOOKS.md) - All hook layers documentation

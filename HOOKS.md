# GoREST Hooks System

A comprehensive hook system for customizing business logic at multiple layers.

## Overview

The hooks system provides **4 distinct layers** for customization:

1. **StateProcessor** - Process state for all write operations (Create/Update/Delete)
2. **SQLQueryListener** - Observe SQL queries (before/after execution)
3. **SQLQueryOverride** - Override default SQL generation
4. **Serializer** - Transform response data before sending to client

## Architecture

All hooks are located in `/internal/hooks/` and are organized by:
- `hooks.go` - Core interfaces and NoOpHooks
- `factory.go` - Centralized hook registry
- `todo.go` - Todo resource hooks implementation
- `user.go` - User resource hooks implementation

## Hook Interfaces

### 1. StateProcessor

Single method for processing ALL write operations:

```go
type StateProcessor[T any] interface {
    StateProcessor(ctx context.Context, operation Operation, id any, model *T) error
}
```

**Operations:**
- `hooks.OperationCreate` - Creating new entity
- `hooks.OperationUpdate` - Updating existing entity
- `hooks.OperationDelete` - Deleting entity (model will be nil)

**Example:**
```go
func (h *TodoHooks) StateProcessor(ctx context.Context, operation hooks.Operation, id any, todo *models.Todo) error {
    switch operation {
    case hooks.OperationCreate:
        // Validate title
        if todo.Title == "" {
            return errors.New("title required")
        }
        // Enrich with user_id from context
        if userID := ctx.Value("user_id"); userID != nil {
            todo.UserId = userID.(*string)
        }

    case hooks.OperationUpdate:
        // Validate and set updated_at
        if todo.Title == "" {
            return errors.New("title required")
        }
        now := time.Now()
        todo.UpdatedAt = &now

    case hooks.OperationDelete:
        // Verify user owns this todo
        if userID := ctx.Value("user_id"); userID != nil {
            // Check ownership from database
        }
    }
    return nil
}
```

### 2. SQLQueryListener

Observe queries before and after execution:

```go
type SQLQueryListener[T any] interface {
    BeforeQuery(ctx context.Context, operation Operation, query string, args []any) (string, []any, error)
    AfterQuery(ctx context.Context, operation Operation, query string, args []any, result any, err error) error
}
```

**Example:**
```go
func (h *TodoHooks) BeforeQuery(ctx context.Context, operation hooks.Operation, query string, args []any) (string, []any, error) {
    log.Printf("[%s] Query: %s, Args: %v", operation, query, args)

    // Could modify query or args here
    return query, args, nil
}

func (h *TodoHooks) AfterQuery(ctx context.Context, operation hooks.Operation, query string, args []any, result any, err error) error {
    if err != nil {
        log.Printf("[%s] FAILED: %v", operation, err)
        // Send to error tracking
    } else {
        log.Printf("[%s] SUCCESS", operation)
        // Record metrics
    }
    return nil
}
```

### 3. SQLQueryOverride

Single method for overriding SQL for any operation:

```go
type SQLQueryOverride[T any] interface {
    OverrideQuery(ctx context.Context, operation Operation, id any, model *T) (query string, args []any, skip bool)
}
```

**Example:**
```go
func (h *TodoHooks) OverrideQuery(ctx context.Context, operation hooks.Operation, id any, todo *models.Todo) (string, []any, bool) {
    switch operation {
    case hooks.OperationGetAll:
        // Custom query with ordering
        query := "SELECT * FROM todo WHERE user_id = $1 ORDER BY created_at DESC"
        args := []any{ctx.Value("user_id")}
        return query, args, true // skip=true to use this query

    case hooks.OperationDelete:
        // Soft delete
        query := "UPDATE todo SET deleted_at = NOW() WHERE id = $1"
        return query, []any{id}, true

    default:
        // Use default implementation
        return "", nil, false
    }
}
```

### 4. Serializer

Transform response data before sending to client:

```go
type Serializer[T any] interface {
    SerializeOne(ctx context.Context, operation Operation, model *T) error
    SerializeMany(ctx context.Context, operation Operation, models *[]T) error
}
```

**Example:**
```go
func (h *TodoHooks) SerializeOne(ctx context.Context, operation hooks.Operation, todo *models.Todo) error {
    // Add computed fields
    // todo.WordCount = len(strings.Fields(todo.Content))

    // Filter sensitive data
    // if !hasAdminRole(ctx) {
    //     todo.InternalNotes = ""
    // }

    return nil
}

func (h *TodoHooks) SerializeMany(ctx context.Context, operation hooks.Operation, todos *[]models.Todo) error {
    // Batch load related data
    // userMap := loadUsersForTodos(todos)

    for i := range *todos {
        h.SerializeOne(ctx, operation, &(*todos)[i])
    }
    return nil
}
```

## Complete Example

File: `/internal/hooks/todo.go`

```go
package hooks

import (
    "context"
    "fmt"
    "github.com/nicolasbonnici/gorest/internal/api/models"
)

type TodoHooks struct {
    NoOpHooks[models.Todo] // Embed defaults
}

// StateProcessor - handles all write operations
func (h *TodoHooks) StateProcessor(ctx context.Context, operation Operation, id any, todo *models.Todo) error {
    switch operation {
    case OperationCreate:
        if todo.Title == "" {
            return fmt.Errorf("title required")
        }
        todo.Title = strings.TrimSpace(todo.Title)

    case OperationUpdate:
        if todo.Title == "" {
            return fmt.Errorf("title required")
        }
        now := time.Now()
        todo.UpdatedAt = &now

    case OperationDelete:
        // Validate delete permissions
        return nil
    }
    return nil
}

// BeforeQuery - log queries
func (h *TodoHooks) BeforeQuery(ctx context.Context, operation Operation, query string, args []any) (string, []any, error) {
    log.Printf("[%s] %s", operation, query)
    return query, args, nil
}

// AfterQuery - log results
func (h *TodoHooks) AfterQuery(ctx context.Context, operation Operation, query string, args []any, result any, err error) error {
    if err != nil {
        log.Printf("[%s] FAILED: %v", operation, err)
    }
    return nil
}

// OverrideQuery - custom SQL
func (h *TodoHooks) OverrideQuery(ctx context.Context, operation Operation, id any, todo *models.Todo) (string, []any, bool) {
    if operation == OperationGetAll {
        query := "SELECT * FROM todo ORDER BY created_at DESC"
        return query, []any{}, true
    }
    return "", nil, false
}

// SerializeOne - transform response
func (h *TodoHooks) SerializeOne(ctx context.Context, operation Operation, todo *models.Todo) error {
    // Add computed fields, filter data
    return nil
}

// SerializeMany - transform list response
func (h *TodoHooks) SerializeMany(ctx context.Context, operation Operation, todos *[]models.Todo) error {
    for i := range *todos {
        h.SerializeOne(ctx, operation, &(*todos)[i])
    }
    return nil
}
```

## Registration

### Method 1: Direct

```go
import (
    "github.com/nicolasbonnici/gorest/internal/crud"
    "github.com/nicolasbonnici/gorest/internal/hooks"
)

func RegisterTodoRoutes(router fiber.Router, db *pgxpool.Pool) {
    res := &TodoResource{
        DB:   db,
        CRUD: crud.NewWithHooks[models.Todo](db, &hooks.TodoHooks{}),
    }
    router.Post("/todos", res.Create)
}
```

### Method 2: Factory (Centralized)

```go
// During app initialization
factory := hooks.NewHookFactory()
factory.Register("todo", &hooks.TodoHooks{})
factory.Register("user", &hooks.UserHooks{})

// In route registration
func RegisterTodoRoutes(router fiber.Router, db *pgxpool.Pool, factory *hooks.HookFactory) {
    h, _ := hooks.GetHooksTyped[models.Todo](factory, "todo")

    res := &TodoResource{
        DB:   db,
        CRUD: crud.NewWithHooks[models.Todo](db, h),
    }
    router.Post("/todos", res.Create)
}
```

## Common Patterns

### Multi-Tenancy
```go
func (h *TodoHooks) OverrideQuery(ctx context.Context, operation hooks.Operation, id any, model *models.Todo) (string, []any, bool) {
    if operation == hooks.OperationGetAll {
        userID := ctx.Value("user_id").(string)
        query := "SELECT * FROM todo WHERE user_id = $1"
        return query, []any{userID}, true
    }
    return "", nil, false
}
```

### Soft Delete
```go
func (h *TodoHooks) OverrideQuery(ctx context.Context, operation hooks.Operation, id any, model *models.Todo) (string, []any, bool) {
    switch operation {
    case hooks.OperationDelete:
        query := "UPDATE todo SET deleted_at = NOW() WHERE id = $1"
        return query, []any{id}, true
    case hooks.OperationGetAll:
        query := "SELECT * FROM todo WHERE deleted_at IS NULL"
        return query, []any{}, true
    }
    return "", nil, false
}
```

### Audit Logging
```go
func (h *TodoHooks) AfterQuery(ctx context.Context, operation hooks.Operation, query string, args []any, result any, err error) error {
    auditLog := AuditLog{
        UserID:    ctx.Value("user_id"),
        Operation: string(operation),
        Success:   err == nil,
        Timestamp: time.Now(),
    }
    saveAuditLog(auditLog)
    return nil
}
```

## Execution Flow

```
HTTP Request
    ↓
Handler
    ↓
CRUD Layer
    ↓
StateProcessor (Create/Update/Delete only)
    ↓
OverrideQuery (optional custom SQL)
    ↓
BeforeQuery (log, modify)
    ↓
Execute SQL
    ↓
AfterQuery (log results)
    ↓
SerializeOne/Many (transform response)
    ↓
HTTP Response
```

## Operations Constants

```go
const (
    OperationCreate  Operation = "CREATE"
    OperationGetAll  Operation = "GET_ALL"
    OperationGetByID Operation = "GET_BY_ID"
    OperationUpdate  Operation = "UPDATE"
    OperationDelete  Operation = "DELETE"
)
```

## Key Benefits

✅ **Simplified Interface** - Single method per hook type
✅ **Operation-based** - Switch on operation type for flexibility
✅ **Type-safe** - Generics ensure compile-time safety
✅ **Testable** - Each hook method is independently testable
✅ **Composable** - Embed NoOpHooks and override only what you need
✅ **Centralized** - All hooks in `/internal/hooks/` folder

## Files Structure

```
/internal/hooks/
├── hooks.go         # Interfaces and NoOpHooks
├── factory.go       # Hook registry and management
├── todo.go          # Todo resource hooks
└── user.go          # User resource hooks
```

## Migration from Old System

**Changes made:**
1. Moved hooks from `/internal/crud/` to `/internal/hooks/`
2. Renamed `ProcessCreate/ProcessUpdate` → `StateProcessor` (single method)
3. Added `StateProcessor` support for Delete operation
4. Renamed `Override{Create,GetAll,GetByID,Update,Delete}` → `OverrideQuery` (single method)
5. Renamed `NormalizeOne/NormalizeMany` → `SerializeOne/SerializeMany`
6. All hook methods now receive `operation` parameter for context

## Testing

```go
func TestTodoHooks_StateProcessor(t *testing.T) {
    hooks := &TodoHooks{}

    // Test Create
    todo := &models.Todo{Title: ""}
    err := hooks.StateProcessor(context.Background(), hooks.OperationCreate, nil, todo)
    if err == nil {
        t.Error("Expected error for empty title")
    }

    // Test with valid title
    todo.Title = "Test"
    err = hooks.StateProcessor(context.Background(), hooks.OperationCreate, nil, todo)
    if err != nil {
        t.Errorf("Unexpected error: %v", err)
    }
}
```

## See Also

- `/internal/hooks/todo.go` - Complete Todo hooks example
- `/internal/hooks/user.go` - User hooks example
- `/internal/crud/crud.go` - CRUD layer integration

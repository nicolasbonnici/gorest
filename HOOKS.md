# GoREST Hooks System

A comprehensive hook system for customizing business logic at multiple layers.

## Overview

The hooks system provides **4 distinct layers** for customization:

1. **StateProcessor** - Process state for all write operations (Create/Update/Delete)
2. **SQLQueryListener** - Observe SQL queries (before/after execution)
3. **SQLQueryBuilderModifier** - Modify queries using the query builder
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

### 3. SQLQueryBuilderModifier

Modify queries using the query builder for type-safe SQL generation:

```go
type SQLQueryBuilderModifier[T any] interface {
    ModifySelectQuery(ctx context.Context, operation Operation, builder *query.SelectBuilder) (*query.SelectBuilder, bool)
    ModifyUpdateQuery(ctx context.Context, operation Operation, id any, model *T, builder *query.UpdateBuilder) (*query.UpdateBuilder, bool)
    ModifyDeleteQuery(ctx context.Context, operation Operation, id any, builder *query.DeleteBuilder) (*query.DeleteBuilder, bool)
}
```

**Example:**
```go
func (h *TodoHooks) ModifySelectQuery(ctx context.Context, operation hooks.Operation, builder *query.SelectBuilder) (*query.SelectBuilder, bool) {
    if operation == hooks.OperationGetAll || operation == hooks.OperationGetByID {
        // Filter by user for multi-tenancy
        if userID := ctx.Value("user_id"); userID != nil {
            builder = builder.Where(query.Eq("user_id", userID))
            return builder, true
        }
    }
    return builder, false
}

func (h *TodoHooks) ModifyUpdateQuery(ctx context.Context, operation hooks.Operation, id any, todo *models.Todo, builder *query.UpdateBuilder) (*query.UpdateBuilder, bool) {
    // Could add additional conditions or modifications
    return builder, false
}

func (h *TodoHooks) ModifyDeleteQuery(ctx context.Context, operation hooks.Operation, id any, builder *query.DeleteBuilder) (*query.DeleteBuilder, bool) {
    // Add additional WHERE conditions to delete queries if needed
    // Note: Soft deletes are better implemented using StateProcessor to prevent deletion
    // or by adding a deleted_at column and filtering in ModifySelectQuery
    return builder, false
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
    "github.com/nicolasbonnici/gorest/internal/models"
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

// ModifySelectQuery - customize SELECT queries
func (h *TodoHooks) ModifySelectQuery(ctx context.Context, operation Operation, builder *query.SelectBuilder) (*query.SelectBuilder, bool) {
    if operation == OperationGetAll {
        // Add default ordering
        builder = builder.OrderBy("created_at", query.DESC)
        return builder, true
    }
    return builder, false
}

// ModifyUpdateQuery - customize UPDATE queries
func (h *TodoHooks) ModifyUpdateQuery(ctx context.Context, operation Operation, id any, todo *models.Todo, builder *query.UpdateBuilder) (*query.UpdateBuilder, bool) {
    return builder, false
}

// ModifyDeleteQuery - customize DELETE queries
func (h *TodoHooks) ModifyDeleteQuery(ctx context.Context, operation Operation, id any, builder *query.DeleteBuilder) (*query.DeleteBuilder, bool) {
    return builder, false
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
func (h *TodoHooks) ModifySelectQuery(ctx context.Context, operation hooks.Operation, builder *query.SelectBuilder) (*query.SelectBuilder, bool) {
    if operation == hooks.OperationGetAll || operation == hooks.OperationGetByID {
        if userID := ctx.Value("user_id"); userID != nil {
            builder = builder.Where(query.Eq("user_id", userID))
            return builder, true
        }
    }
    return builder, false
}
```

### Soft Delete
```go
// Filter out soft-deleted records in queries
func (h *TodoHooks) ModifySelectQuery(ctx context.Context, operation hooks.Operation, builder *query.SelectBuilder) (*query.SelectBuilder, bool) {
    if operation == hooks.OperationGetAll || operation == hooks.OperationGetByID {
        builder = builder.Where(query.IsNull("deleted_at"))
        return builder, true
    }
    return builder, false
}

// Prevent actual deletion by returning an error
func (h *TodoHooks) StateProcessor(ctx context.Context, operation hooks.Operation, id any, model *models.Todo) error {
    if operation == hooks.OperationDelete {
        // Instead of deleting, update the deleted_at timestamp
        // This requires a separate update operation
        return errors.New("soft delete should be implemented via update operation")
    }
    return nil
}

// Note: For a complete soft delete implementation, you would create a
// SoftDelete method in your resource that updates deleted_at instead of calling Delete
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
Query Builder Construction
    ↓
ModifySelectQuery/ModifyUpdateQuery/ModifyDeleteQuery (optional modifications)
    ↓
Build SQL with parameterization
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

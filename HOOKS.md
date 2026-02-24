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

## Testing Patterns for Hook System

This section documents comprehensive testing patterns for the hooks system, providing guidance for contributors.

### Table-Driven Test Pattern

All hook tests should follow the table-driven pattern for maintainability and coverage:

```go
func TestNoOpHooks_StateProcessor(t *testing.T) {
    hooks := NoOpHooks[testModel]{}
    model := &testModel{ID: "1", Name: "Test"}

    operations := []Operation{
        OperationCreate,
        OperationGetAll,
        OperationGetByID,
        OperationUpdate,
        OperationDelete,
    }

    for _, op := range operations {
        t.Run(string(op), func(t *testing.T) {
            err := hooks.StateProcessor(context.Background(), op, "1", model)
            if err != nil {
                t.Errorf("Expected nil error, got %v", err)
            }
        })
    }
}
```

### Testing StateProcessor

Test all operations (Create, Update, Delete) with valid and invalid inputs:

```go
func TestTodoHooks_StateProcessor_Validation(t *testing.T) {
    hooks := &TodoHooks{}

    tests := []struct {
        name      string
        operation Operation
        todo      *models.Todo
        expectErr bool
    }{
        {
            name:      "valid create",
            operation: OperationCreate,
            todo:      &models.Todo{Title: "Valid Title"},
            expectErr: false,
        },
        {
            name:      "invalid create - empty title",
            operation: OperationCreate,
            todo:      &models.Todo{Title: ""},
            expectErr: true,
        },
        {
            name:      "valid update",
            operation: OperationUpdate,
            todo:      &models.Todo{ID: "1", Title: "Updated"},
            expectErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := hooks.StateProcessor(context.Background(), tt.operation, tt.todo.ID, tt.todo)
            if (err != nil) != tt.expectErr {
                t.Errorf("Expected error=%v, got error=%v", tt.expectErr, err)
            }
        })
    }
}
```

### Testing SQLQueryListener

Test both BeforeQuery and AfterQuery with various scenarios:

```go
func TestHooks_BeforeQuery(t *testing.T) {
    hooks := &TodoHooks{}

    tests := []struct {
        name          string
        operation     Operation
        query         string
        args          []any
        expectedQuery string
        expectedArgs  []any
    }{
        {
            name:          "simple query",
            operation:     OperationGetAll,
            query:         "SELECT * FROM todos",
            args:          []any{},
            expectedQuery: "SELECT * FROM todos",
            expectedArgs:  []any{},
        },
        {
            name:          "query with args",
            operation:     OperationGetByID,
            query:         "SELECT * FROM todos WHERE id = $1",
            args:          []any{"123"},
            expectedQuery: "SELECT * FROM todos WHERE id = $1",
            expectedArgs:  []any{"123"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resultQuery, resultArgs, err := hooks.BeforeQuery(
                context.Background(),
                tt.operation,
                tt.query,
                tt.args,
            )

            if err != nil {
                t.Errorf("Expected nil error, got %v", err)
            }

            if resultQuery != tt.expectedQuery {
                t.Errorf("Expected query %s, got %s", tt.expectedQuery, resultQuery)
            }

            if len(resultArgs) != len(tt.expectedArgs) {
                t.Errorf("Expected %d args, got %d", len(tt.expectedArgs), len(resultArgs))
            }
        })
    }
}
```

### Testing SQLQueryBuilderModifier

Test query modifications return correct modified flag:

```go
func TestTodoHooks_ModifySelectQuery(t *testing.T) {
    hooks := &TodoHooks{}

    tests := []struct {
        name            string
        operation       Operation
        expectModified  bool
    }{
        {
            name:           "modifies GetAll",
            operation:      OperationGetAll,
            expectModified: true,
        },
        {
            name:           "does not modify GetByID",
            operation:      OperationGetByID,
            expectModified: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            builder := &query.SelectBuilder{}
            resultBuilder, modified := hooks.ModifySelectQuery(
                context.Background(),
                tt.operation,
                builder,
            )

            if modified != tt.expectModified {
                t.Errorf("Expected modified=%v, got %v", tt.expectModified, modified)
            }

            if modified && resultBuilder == nil {
                t.Error("Expected non-nil builder when modified=true")
            }
        })
    }
}
```

### Testing Serializer

Test both single and multiple model serialization:

```go
func TestHooks_SerializeOne(t *testing.T) {
    hooks := &TodoHooks{}
    model := &models.Todo{ID: "1", Title: "Test"}

    operations := []Operation{
        OperationCreate,
        OperationGetByID,
        OperationUpdate,
    }

    for _, op := range operations {
        t.Run(string(op), func(t *testing.T) {
            err := hooks.SerializeOne(context.Background(), op, model)
            if err != nil {
                t.Errorf("Expected nil error, got %v", err)
            }
        })
    }
}

func TestHooks_SerializeMany(t *testing.T) {
    hooks := &TodoHooks{}

    tests := []struct {
        name   string
        models *[]models.Todo
    }{
        {
            name:   "empty slice",
            models: &[]models.Todo{},
        },
        {
            name: "single model",
            models: &[]models.Todo{
                {ID: "1", Title: "Test1"},
            },
        },
        {
            name: "multiple models",
            models: &[]models.Todo{
                {ID: "1", Title: "Test1"},
                {ID: "2", Title: "Test2"},
                {ID: "3", Title: "Test3"},
            },
        },
        {
            name:   "nil slice",
            models: nil,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := hooks.SerializeMany(context.Background(), OperationGetAll, tt.models)
            if err != nil {
                t.Errorf("Expected nil error, got %v", err)
            }
        })
    }
}
```

### Testing Hook Factory

Test factory operations with comprehensive edge cases:

```go
func TestHookFactory_Register(t *testing.T) {
    tests := []struct {
        name         string
        resourceName string
        hooks        interface{}
        expectStored bool
    }{
        {
            name:         "simple registration",
            resourceName: "todos",
            hooks:        &TodoHooks{},
            expectStored: true,
        },
        {
            name:         "overwrite existing",
            resourceName: "todos",
            hooks:        &NoOpHooks[models.Todo]{},
            expectStored: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            factory := NewHookFactory()
            factory.Register(tt.resourceName, tt.hooks)

            _, exists := factory.GetHooks(tt.resourceName)
            if exists != tt.expectStored {
                t.Errorf("Expected exists=%v, got %v", tt.expectStored, exists)
            }
        })
    }
}

func TestGetHooksTyped(t *testing.T) {
    tests := []struct {
        name            string
        resourceName    string
        setupFunc       func(*HookFactory)
        expectError     bool
        expectNoOpHooks bool
    }{
        {
            name:         "get typed hooks successfully",
            resourceName: "todos",
            setupFunc: func(f *HookFactory) {
                f.Register("todos", &TodoHooks{})
            },
            expectError:     false,
            expectNoOpHooks: false,
        },
        {
            name:            "non-existent resource returns NoOpHooks",
            resourceName:    "missing",
            setupFunc:       func(f *HookFactory) {},
            expectError:     false,
            expectNoOpHooks: true,
        },
        {
            name:         "wrong type returns error",
            resourceName: "todos",
            setupFunc: func(f *HookFactory) {
                f.Register("todos", &NoOpHooks[string]{}) // Wrong type
            },
            expectError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            factory := NewHookFactory()
            tt.setupFunc(factory)

            hooks, err := GetHooksTyped[models.Todo](factory, tt.resourceName)

            if tt.expectError && err == nil {
                t.Error("Expected error for type mismatch")
            }
            if !tt.expectError && err != nil {
                t.Errorf("Unexpected error: %v", err)
            }
            if !tt.expectError && hooks == nil {
                t.Fatal("Expected non-nil hooks")
            }
        })
    }
}
```

### Testing with Context

Test context-aware hooks with user data, tenant IDs, etc.:

```go
func TestHooks_WithContext(t *testing.T) {
    hooks := &TodoHooks{}

    tests := []struct {
        name    string
        ctx     context.Context
        wantErr bool
    }{
        {
            name: "with user context",
            ctx: context.WithValue(context.Background(), "user_id", "user123"),
            wantErr: false,
        },
        {
            name: "with tenant context",
            ctx: context.WithValue(context.Background(), "tenant_id", "tenant456"),
            wantErr: false,
        },
        {
            name: "empty context",
            ctx: context.Background(),
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            model := &models.Todo{Title: "Test"}
            err := hooks.StateProcessor(tt.ctx, OperationCreate, nil, model)

            if (err != nil) != tt.wantErr {
                t.Errorf("Expected error=%v, got error=%v", tt.wantErr, err)
            }
        })
    }
}
```

### Testing Generic Type Support

Test hooks work with different generic types:

```go
func TestNoOpHooks_DifferentTypes(t *testing.T) {
    t.Run("string type", func(t *testing.T) {
        hooks := NoOpHooks[string]{}
        value := "test"
        err := hooks.StateProcessor(context.Background(), OperationCreate, "1", &value)
        if err != nil {
            t.Errorf("Expected nil error, got %v", err)
        }
    })

    t.Run("int type", func(t *testing.T) {
        hooks := NoOpHooks[int]{}
        value := 42
        err := hooks.StateProcessor(context.Background(), OperationCreate, "1", &value)
        if err != nil {
            t.Errorf("Expected nil error, got %v", err)
        }
    })

    t.Run("struct type", func(t *testing.T) {
        hooks := NoOpHooks[models.Todo]{}
        value := models.Todo{ID: "1", Title: "Test"}
        err := hooks.StateProcessor(context.Background(), OperationCreate, "1", &value)
        if err != nil {
            t.Errorf("Expected nil error, got %v", err)
        }
    })
}
```

### Testing Interface Implementation

Verify hooks implement all required interfaces:

```go
func TestHooks_ImplementsInterfaces(t *testing.T) {
    // Compile-time interface compliance checks
    var _ Hooks[models.Todo] = &TodoHooks{}
    var _ StateProcessor[models.Todo] = &TodoHooks{}
    var _ SQLQueryListener[models.Todo] = &TodoHooks{}
    var _ SQLQueryBuilderModifier[models.Todo] = &TodoHooks{}
    var _ Serializer[models.Todo] = &TodoHooks{}
}

func TestNoOpHooks_ImplementsInterfaces(t *testing.T) {
    // Verify NoOpHooks implements all interfaces
    var _ Hooks[models.Todo] = NoOpHooks[models.Todo]{}
    var _ StateProcessor[models.Todo] = NoOpHooks[models.Todo]{}
    var _ SQLQueryListener[models.Todo] = NoOpHooks[models.Todo]{}
    var _ SQLQueryBuilderModifier[models.Todo] = NoOpHooks[models.Todo]{}
    var _ Serializer[models.Todo] = NoOpHooks[models.Todo]{}
}
```

### Integration Testing

Test hooks with actual CRUD operations:

```go
//go:build integration

func TestHooks_Integration(t *testing.T) {
    db := testhelpers.SetupPostgres(t)
    defer db.Close()

    hooks := &TodoHooks{}
    todoCRUD := crud.New[models.Todo](db, hooks)

    // Test Create with hooks
    todo := &models.Todo{Title: "Integration Test"}
    created, err := todoCRUD.Create(context.Background(), todo)
    if err != nil {
        t.Fatalf("Create failed: %v", err)
    }

    // Verify hooks were applied
    if created.Title != "Integration Test" {
        t.Errorf("Expected title 'Integration Test', got '%s'", created.Title)
    }

    // Test GetByID with hooks
    fetched, err := todoCRUD.GetByID(context.Background(), created.ID)
    if err != nil {
        t.Fatalf("GetByID failed: %v", err)
    }

    if fetched.ID != created.ID {
        t.Errorf("Expected ID %s, got %s", created.ID, fetched.ID)
    }
}
```

### Coverage Best Practices

1. **Test all operations**: Create, GetAll, GetByID, Update, Delete
2. **Test edge cases**: nil models, empty slices, invalid IDs
3. **Test context handling**: with and without context values
4. **Test error conditions**: validation failures, database errors
5. **Test type safety**: different generic types, type mismatches
6. **Test factory operations**: registration, retrieval, clearing
7. **Test interface compliance**: compile-time checks
8. **Aim for 90%+ coverage**: Critical business logic should be well-tested

### Running Hook Tests

```bash
# Run all hook tests
go test -v ./hooks

# Run with coverage
go test -cover ./hooks

# Run specific test
go test -v ./hooks -run TestHookFactory_Register

# Run integration tests
go test -tags=integration -v ./hooks
```

### Test File Organization

```
/hooks/
├── hooks.go           # Hook interfaces and NoOpHooks
├── factory.go         # Hook factory
├── hooks_test.go      # NoOpHooks tests
├── factory_test.go    # Factory tests
└── integration_test.go # Integration tests (//go:build integration)
```

## See Also

- `/internal/hooks/todo.go` - Complete Todo hooks example
- `/internal/hooks/user.go` - User hooks example
- `/internal/crud/crud.go` - CRUD layer integration
- `hooks/hooks_test.go` - Comprehensive test examples
- `hooks/factory_test.go` - Factory test patterns

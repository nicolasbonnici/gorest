# GoREST Hooks System Guide

This guide explains the comprehensive hook system implemented in GoREST that allows you to override and customize business logic for CRUD operations at multiple layers.

## Overview

The hook system provides **4 distinct layers** of customization:

1. **State Processor** - Validates and transforms data before write operations
2. **SQL Query Listener** - Observes SQL queries for logging, metrics, and audit
3. **SQL Query Override** - Provides custom SQL query generation
4. **Resource Normalizer** - Transforms resources before sending to client

## Architecture

```
HTTP Request
    ↓
Handler (Resource Layer)
    ↓
CRUD Layer with Hooks
    ↓
┌─────────────────────────────────────────────────┐
│  Hook Execution Flow:                           │
│                                                  │
│  1. StateProcessor.ProcessCreate/Update         │
│     - Validate input                             │
│     - Transform/enrich data                      │
│     - Business rule enforcement                  │
│                                                  │
│  2. SQLQueryOverride.Override*                   │
│     - Generate custom SQL (optional)             │
│     - Add WHERE clauses, JOINs, etc.            │
│                                                  │
│  3. SQLQueryListener.BeforeQuery                 │
│     - Log query                                  │
│     - Modify query/args                          │
│     - Record metrics                             │
│                                                  │
│  4. Execute SQL Query                            │
│                                                  │
│  5. SQLQueryListener.AfterQuery                  │
│     - Log results                                │
│     - Record metrics                             │
│     - Audit trail                                │
│                                                  │
│  6. ResourceNormalizer.NormalizeOne/Many         │
│     - Add computed fields                        │
│     - Filter sensitive data                      │
│     - Transform response format                  │
└─────────────────────────────────────────────────┘
    ↓
HTTP Response
```

## Hook Interfaces

### 1. StateProcessor

Processes and validates entity state before write operations.

```go
type StateProcessor[T any] interface {
    ProcessCreate(ctx context.Context, model *T) error
    ProcessUpdate(ctx context.Context, id any, model *T) error
}
```

**Use Cases:**
- Input validation
- Data normalization (trim, lowercase, etc.)
- Business rule enforcement
- Data enrichment from context (user_id, timestamps)
- Authorization checks

**Example:**
```go
func (h *TodoHooks) ProcessCreate(ctx context.Context, todo *models.Todo) error {
    // Validation
    if strings.TrimSpace(todo.Title) == "" {
        return fmt.Errorf("title is required")
    }

    // Normalization
    todo.Title = strings.TrimSpace(todo.Title)

    // Enrichment
    if userID := ctx.Value("user_id"); userID != nil {
        todo.UserId = userID.(*string)
    }

    return nil
}
```

### 2. SQLQueryListener

Observes SQL query execution for monitoring and audit purposes.

```go
type SQLQueryListener[T any] interface {
    BeforeQuery(ctx context.Context, operation string, query string, args []any) (string, []any, error)
    AfterQuery(ctx context.Context, operation string, query string, args []any, result any, err error) error
}
```

**Use Cases:**
- Query logging
- Performance metrics
- Audit trails
- Query modification (add timeout hints, etc.)
- Error tracking integration

**Example:**
```go
func (h *TodoHooks) BeforeQuery(ctx context.Context, operation string, query string, args []any) (string, []any, error) {
    log.Printf("[%s] Query: %s, Args: %v", operation, query, args)

    // Could modify query here, e.g., add query hints
    // modifiedQuery := "/*+ MAX_EXECUTION_TIME(5000) */ " + query

    return query, args, nil
}

func (h *TodoHooks) AfterQuery(ctx context.Context, operation string, query string, args []any, result any, err error) error {
    if err != nil {
        log.Printf("[%s] FAILED: %v", operation, err)
        // Send to error tracking service
        return nil
    }

    log.Printf("[%s] SUCCESS", operation)
    // Record metrics
    // metrics.RecordQuery(operation, time.Since(start))

    return nil
}
```

### 3. SQLQueryOverride

Provides complete control over SQL query generation.

```go
type SQLQueryOverride[T any] interface {
    OverrideCreate(ctx context.Context, model *T) (query string, args []any, skip bool)
    OverrideGetAll(ctx context.Context) (query string, args []any, skip bool)
    OverrideGetByID(ctx context.Context, id any) (query string, args []any, skip bool)
    OverrideUpdate(ctx context.Context, id any, model *T) (query string, args []any, skip bool)
    OverrideDelete(ctx context.Context, id any) (query string, args []any, skip bool)
}
```

**Use Cases:**
- Custom ordering (ORDER BY)
- Complex filtering (multi-tenant, soft delete)
- JOINs with related tables
- Pagination
- Optimistic locking
- Custom RETURNING clauses

**Example:**
```go
func (h *TodoHooks) OverrideGetAll(ctx context.Context) (query string, args []any, skip bool) {
    query = `
        SELECT id, user_id, title, content, updated_at, created_at
        FROM todo
        WHERE deleted_at IS NULL
    `
    args = []any{}

    // Add user filtering for multi-tenancy
    if userID := ctx.Value("user_id"); userID != nil {
        query += " AND user_id = $1"
        args = append(args, userID)
    }

    // Add ordering
    query += " ORDER BY created_at DESC"

    return query, args, true // skip=true means use this custom query
}

func (h *TodoHooks) OverrideDelete(ctx context.Context, id any) (query string, args []any, skip bool) {
    // Soft delete instead of hard delete
    query = "UPDATE todo SET deleted_at = $1 WHERE id = $2"
    args = []any{time.Now(), id}

    return query, args, true
}
```

### 4. ResourceNormalizer

Transforms resources before sending to the client.

```go
type ResourceNormalizer[T any] interface {
    NormalizeOne(ctx context.Context, model *T) error
    NormalizeMany(ctx context.Context, models *[]T) error
}
```

**Use Cases:**
- Add computed fields
- Format dates/times
- Add metadata
- Filter sensitive data
- Transform to DTOs
- Add HATEOAS links

**Example:**
```go
func (h *TodoHooks) NormalizeOne(ctx context.Context, todo *models.Todo) error {
    // Add computed field (requires extending model or using DTO)
    // todo.WordCount = len(strings.Fields(todo.Content))

    // Format dates
    if todo.CreatedAt != nil {
        // Could add formatted version to response
        log.Printf("Created: %s", todo.CreatedAt.Format(time.RFC3339))
    }

    // Filter sensitive data based on permissions
    if !hasAdminRole(ctx) {
        // todo.InternalNotes = ""
    }

    return nil
}

func (h *TodoHooks) NormalizeMany(ctx context.Context, todos *[]models.Todo) error {
    // Batch load related data for efficiency
    // userMap := loadUsersForTodos(todos)

    for i := range *todos {
        if err := h.NormalizeOne(ctx, &(*todos)[i]); err != nil {
            return err
        }
    }

    return nil
}
```

## Complete Hook Implementation

Here's a complete example implementing all hook interfaces:

```go
package resources

import (
    "context"
    "fmt"
    "log"
    "strings"
    "time"

    "github.com/nicolasbonnici/gorest/internal/api/models"
    "github.com/nicolasbonnici/gorest/internal/crud"
)

type TodoHooks struct {
    crud.NoOpHooks[models.Todo]
}

// StateProcessor
func (h *TodoHooks) ProcessCreate(ctx context.Context, todo *models.Todo) error {
    if strings.TrimSpace(todo.Title) == "" {
        return fmt.Errorf("title is required")
    }
    todo.Title = strings.TrimSpace(todo.Title)
    return nil
}

func (h *TodoHooks) ProcessUpdate(ctx context.Context, id any, todo *models.Todo) error {
    if strings.TrimSpace(todo.Title) == "" {
        return fmt.Errorf("title is required")
    }
    todo.Title = strings.TrimSpace(todo.Title)
    now := time.Now()
    todo.UpdatedAt = &now
    return nil
}

// SQLQueryListener
func (h *TodoHooks) BeforeQuery(ctx context.Context, operation string, query string, args []any) (string, []any, error) {
    log.Printf("[%s] Query: %s", operation, query)
    return query, args, nil
}

func (h *TodoHooks) AfterQuery(ctx context.Context, operation string, query string, args []any, result any, err error) error {
    if err != nil {
        log.Printf("[%s] FAILED: %v", operation, err)
    } else {
        log.Printf("[%s] SUCCESS", operation)
    }
    return nil
}

// SQLQueryOverride
func (h *TodoHooks) OverrideGetAll(ctx context.Context) (query string, args []any, skip bool) {
    query = "SELECT id, user_id, title, content, updated_at, created_at FROM todo ORDER BY created_at DESC"
    return query, []any{}, true
}

// ResourceNormalizer
func (h *TodoHooks) NormalizeOne(ctx context.Context, todo *models.Todo) error {
    log.Printf("Normalizing todo: %s", todo.Id)
    return nil
}

func (h *TodoHooks) NormalizeMany(ctx context.Context, todos *[]models.Todo) error {
    for i := range *todos {
        h.NormalizeOne(ctx, &(*todos)[i])
    }
    return nil
}
```

## Registering Hooks

### Method 1: Direct Registration

```go
func RegisterTodoRoutes(router fiber.Router, db *pgxpool.Pool, jwtSecret string) {
    res := &TodoResource{
        DB:   db,
        CRUD: crud.NewWithHooks[models.Todo](db, &TodoHooks{}),
    }
    router.Post("/todos", res.Create)
}
```

### Method 2: Using Hook Factory (Recommended)

```go
// In initialization code
factory := crud.NewHookFactory()
factory.Register("todo", &TodoHooks{})
factory.Register("user", &UserHooks{})

// In route registration
func RegisterTodoRoutes(router fiber.Router, db *pgxpool.Pool, factory *crud.HookFactory) {
    hooks, _ := crud.GetHooksTyped[models.Todo](factory, "todo")

    res := &TodoResource{
        DB:   db,
        CRUD: crud.NewWithHooks[models.Todo](db, hooks),
    }
    router.Post("/todos", res.Create)
}
```

### Method 3: Global Factory

```go
// During app initialization
crud.RegisterGlobal("todo", &TodoHooks{})

// In route registration
func RegisterTodoRoutes(router fiber.Router, db *pgxpool.Pool) {
    hooksInterface, _ := crud.GetGlobal("todo")
    hooks := hooksInterface.(crud.Hooks[models.Todo])

    res := &TodoResource{
        DB:   db,
        CRUD: crud.NewWithHooks[models.Todo](db, hooks),
    }
    router.Post("/todos", res.Create)
}
```

## Common Patterns

### Multi-Tenancy

```go
func (h *TodoHooks) OverrideGetAll(ctx context.Context) (query string, args []any, skip bool) {
    query = "SELECT * FROM todo WHERE user_id = $1 ORDER BY created_at DESC"

    userID := ctx.Value("user_id").(string)
    args = []any{userID}

    return query, args, true
}
```

### Soft Delete

```go
func (h *TodoHooks) OverrideDelete(ctx context.Context, id any) (query string, args []any, skip bool) {
    query = "UPDATE todo SET deleted_at = $1 WHERE id = $2"
    args = []any{time.Now(), id}
    return query, args, true
}

func (h *TodoHooks) OverrideGetAll(ctx context.Context) (query string, args []any, skip bool) {
    query = "SELECT * FROM todo WHERE deleted_at IS NULL"
    return query, []any{}, true
}
```

### Audit Logging

```go
func (h *TodoHooks) AfterQuery(ctx context.Context, operation string, query string, args []any, result any, err error) error {
    userID := ctx.Value("user_id")

    auditLog := AuditLog{
        UserID:    userID,
        Operation: operation,
        Query:     query,
        Success:   err == nil,
        Timestamp: time.Now(),
    }

    // Save to audit table
    saveAuditLog(auditLog)

    return nil
}
```

### Computed Fields

```go
type TodoDTO struct {
    models.Todo
    WordCount    int    `json:"word_count"`
    IsRecent     bool   `json:"is_recent"`
    RelativeTime string `json:"relative_time"`
}

func (h *TodoHooks) NormalizeOne(ctx context.Context, todo *models.Todo) error {
    // In production, you'd convert to DTO
    // This is conceptual example
    wordCount := len(strings.Fields(todo.Content))
    isRecent := todo.CreatedAt.After(time.Now().Add(-24 * time.Hour))

    log.Printf("Todo words: %d, recent: %v", wordCount, isRecent)

    return nil
}
```

### Optimistic Locking

```go
func (h *TodoHooks) OverrideUpdate(ctx context.Context, id any, todo *models.Todo) (query string, args []any, skip bool) {
    query = `
        UPDATE todo
        SET title = $1, content = $2, version = version + 1, updated_at = $3
        WHERE id = $4 AND version = $5
    `

    args = []any{
        todo.Title,
        todo.Content,
        time.Now(),
        id,
        todo.Version, // Assume Todo has Version field
    }

    return query, args, true
}
```

## Selective Hook Override

You don't need to implement all hooks. Use `NoOpHooks` as base and override only what you need:

```go
type TodoHooks struct {
    crud.NoOpHooks[models.Todo] // Provides default no-op implementations
}

// Only override what you need
func (h *TodoHooks) ProcessCreate(ctx context.Context, todo *models.Todo) error {
    // Your custom logic
    return nil
}

// All other hooks will use NoOpHooks default (do nothing)
```

## Testing Hooks

```go
func TestTodoHooks_ProcessCreate(t *testing.T) {
    hooks := &TodoHooks{}

    tests := []struct {
        name    string
        todo    *models.Todo
        wantErr bool
    }{
        {
            name:    "valid todo",
            todo:    &models.Todo{Title: "Test", Content: "Content"},
            wantErr: false,
        },
        {
            name:    "empty title",
            todo:    &models.Todo{Title: "", Content: "Content"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := hooks.ProcessCreate(context.Background(), tt.todo)
            if (err != nil) != tt.wantErr {
                t.Errorf("ProcessCreate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Performance Considerations

1. **Async Operations**: Use goroutines for non-critical operations in AfterQuery
   ```go
   func (h *TodoHooks) AfterQuery(...) error {
       go func() {
           sendNotification()
       }()
       return nil
   }
   ```

2. **Context Timeouts**: Add timeouts for external services
   ```go
   func (h *TodoHooks) ProcessCreate(ctx context.Context, todo *models.Todo) error {
       ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
       defer cancel()

       return externalValidator.Validate(ctx, todo)
   }
   ```

3. **Batch Operations**: Load related data in batches in NormalizeMany
   ```go
   func (h *TodoHooks) NormalizeMany(ctx context.Context, todos *[]models.Todo) error {
       // Load all users in one query
       userIDs := extractUserIDs(todos)
       users := loadUsersBatch(userIDs)

       // Enrich each todo
       for i := range *todos {
           (*todos)[i].User = users[(*todos)[i].UserId]
       }
       return nil
   }
   ```

## Best Practices

1. **Use NoOpHooks as Base**: Always embed `crud.NoOpHooks[T]` to get default implementations
2. **Log Hook Activity**: Use structured logging to track hook execution
3. **Return Errors Properly**: Don't swallow errors in StateProcessor
4. **Keep AfterQuery Fast**: Don't block response on non-critical operations
5. **Test Hooks Separately**: Unit test hooks independently of CRUD layer
6. **Use Context Values**: Pass user info, request ID, etc. via context
7. **Document Custom Queries**: Comment why you're overriding default SQL
8. **Consider DTOs**: For complex normalization, use DTOs instead of modifying models

## Files Created

- `/internal/crud/hooks.go` - Hook interfaces and NoOpHooks
- `/internal/crud/hook_factory.go` - Centralized hook management
- `/internal/api/resources/todo_hooks.go` - Complete Todo hooks example
- `/internal/crud/crud.go` - Updated CRUD layer with hook integration

## Example: Todo Resource with All Hooks

See `/internal/api/resources/todo_hooks.go` for a complete, production-ready example implementing all 4 hook layers with comprehensive comments and logging.

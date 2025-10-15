# GoREST Hooks System - Quick Reference

## 4 Hook Layers Implemented

### 1️⃣ StateProcessor - Write Operation Validation & Transformation
```go
ProcessCreate(ctx context.Context, model *T) error
ProcessUpdate(ctx context.Context, id any, model *T) error
```
**When:** Before any database operation (Create/Update only)
**Purpose:** Validate input, normalize data, enrich from context
**Example:** Trim whitespace, check required fields, inject user_id from JWT

### 2️⃣ SQLQueryListener - Query Observation & Monitoring
```go
BeforeQuery(ctx, operation, query, args) (string, []any, error)
AfterQuery(ctx, operation, query, args, result, err) error
```
**When:** Before and after every SQL query
**Purpose:** Logging, metrics, audit trails
**Example:** Log all queries, send metrics to monitoring, track slow queries

### 3️⃣ SQLQueryOverride - Custom SQL Generation
```go
OverrideCreate(ctx, model) (query, args, skip)
OverrideGetAll(ctx) (query, args, skip)
OverrideGetByID(ctx, id) (query, args, skip)
OverrideUpdate(ctx, id, model) (query, args, skip)
OverrideDelete(ctx, id) (query, args, skip)
```
**When:** Instead of default SQL generation
**Purpose:** Custom queries, filtering, ordering, JOINs, soft delete
**Example:** Add `ORDER BY created_at DESC`, filter by tenant, soft delete

### 4️⃣ ResourceNormalizer - Response Transformation
```go
NormalizeOne(ctx context.Context, model *T) error
NormalizeMany(ctx context.Context, models *[]T) error
```
**When:** After fetching from database, before returning to client
**Purpose:** Add computed fields, format dates, filter sensitive data
**Example:** Add word count, relative timestamps, HATEOAS links

## Execution Order

```
HTTP POST /todos
    ↓
1. StateProcessor.ProcessCreate()     [Validate & enrich]
    ↓
2. SQLQueryOverride.OverrideCreate()  [Custom SQL or default]
    ↓
3. SQLQueryListener.BeforeQuery()     [Log query]
    ↓
4. Execute SQL
    ↓
5. SQLQueryListener.AfterQuery()      [Log result]
    ↓
6. ResourceNormalizer.NormalizeOne()  [Transform response]
    ↓
HTTP Response
```

## Todo Hooks Example

File: `/internal/api/resources/todo_hooks.go`

```go
type TodoHooks struct {
    crud.NoOpHooks[models.Todo]
}

// Layer 1: Validate title is required
func (h *TodoHooks) ProcessCreate(ctx, todo) error {
    if todo.Title == "" {
        return errors.New("title required")
    }
    todo.Title = strings.TrimSpace(todo.Title)
    return nil
}

// Layer 2: Log all queries
func (h *TodoHooks) BeforeQuery(ctx, op, query, args) (string, []any, error) {
    log.Printf("[%s] %s", op, query)
    return query, args, nil
}

// Layer 3: Add ORDER BY to GetAll
func (h *TodoHooks) OverrideGetAll(ctx) (query, args, skip) {
    query = "SELECT * FROM todo ORDER BY created_at DESC"
    return query, []any{}, true
}

// Layer 4: Add computed fields
func (h *TodoHooks) NormalizeOne(ctx, todo) error {
    // Add word count, relative time, etc.
    return nil
}
```

## Registration

```go
// Method 1: Direct
res := &TodoResource{
    DB:   db,
    CRUD: crud.NewWithHooks[models.Todo](db, &TodoHooks{}),
}

// Method 2: Factory (Recommended)
factory := crud.NewHookFactory()
factory.Register("todo", &TodoHooks{})
hooks, _ := crud.GetHooksTyped[models.Todo](factory, "todo")
res := &TodoResource{
    DB:   db,
    CRUD: crud.NewWithHooks[models.Todo](db, hooks),
}
```

## Common Use Cases

| Use Case | Hook Layer | Example |
|----------|------------|---------|
| Validate required fields | StateProcessor | `ProcessCreate()` check title != "" |
| Auto-capitalize title | StateProcessor | `ProcessCreate()` uppercase first char |
| Inject user_id from JWT | StateProcessor | `ProcessCreate()` set from context |
| Log all queries | SQLQueryListener | `BeforeQuery()` log to file |
| Track slow queries | SQLQueryListener | `AfterQuery()` if duration > 1s |
| Multi-tenant filtering | SQLQueryOverride | `OverrideGetAll()` add WHERE tenant_id |
| Soft delete | SQLQueryOverride | `OverrideDelete()` UPDATE deleted_at |
| Custom ordering | SQLQueryOverride | `OverrideGetAll()` ORDER BY created_at |
| Add word count | ResourceNormalizer | `NormalizeOne()` compute words |
| Filter by permissions | ResourceNormalizer | `NormalizeOne()` remove admin fields |
| Batch load users | ResourceNormalizer | `NormalizeMany()` load related data |

## Files Modified/Created

### New Files
- ✅ `/internal/crud/hooks.go` - Hook interfaces
- ✅ `/internal/crud/hook_factory.go` - Centralized hook registry
- ✅ `/internal/api/resources/todo_hooks.go` - Complete Todo example
- ✅ `/HOOKS_GUIDE.md` - Comprehensive guide
- ✅ `/HOOKS_SUMMARY.md` - This quick reference

### Modified Files
- ✅ `/internal/crud/crud.go` - Integrated all 4 hook layers
- ✅ `/internal/api/resources/todo.go` - Enabled hooks for Todo resource

## Testing

```bash
# Build to verify compilation
go build ./...

# Run tests (create tests as needed)
go test ./internal/crud -v
go test ./internal/api/resources -v
```

## Next Steps

1. **Add hooks to other resources** (User, etc.)
2. **Create reusable hook libraries** (validation, audit, etc.)
3. **Add hook configuration** (enable/disable via config)
4. **Implement hook middleware** (chain multiple hooks)
5. **Add metrics integration** (Prometheus, etc.)
6. **Create hook templates** for common patterns

## Benefits

✅ **Separation of Concerns** - Business logic separate from CRUD
✅ **Reusability** - Hooks can be shared across resources
✅ **Testability** - Each hook can be unit tested independently
✅ **Flexibility** - Override only what you need
✅ **Type Safety** - Generic interfaces ensure compile-time checking
✅ **Performance** - Minimal overhead, async operations supported
✅ **Observability** - Built-in logging and metrics support

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    HTTP Request                          │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              Resource Handler Layer                      │
│  • Parse request body                                    │
│  • Call CRUD methods                                     │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                   CRUD Layer                             │
│  ┌─────────────────────────────────────────────────┐   │
│  │  Hook Layer 1: StateProcessor                    │   │
│  │  • ProcessCreate/Update                          │   │
│  │  • Validation, normalization, enrichment         │   │
│  └─────────────────────────────────────────────────┘   │
│                     ↓                                    │
│  ┌─────────────────────────────────────────────────┐   │
│  │  Hook Layer 2: SQLQueryOverride                  │   │
│  │  • Override* methods                             │   │
│  │  • Custom SQL generation (optional)              │   │
│  └─────────────────────────────────────────────────┘   │
│                     ↓                                    │
│  ┌─────────────────────────────────────────────────┐   │
│  │  Hook Layer 3: SQLQueryListener                  │   │
│  │  • BeforeQuery (log, modify)                     │   │
│  └─────────────────────────────────────────────────┘   │
│                     ↓                                    │
│  ┌─────────────────────────────────────────────────┐   │
│  │         Execute SQL Query                        │   │
│  └─────────────────────────────────────────────────┘   │
│                     ↓                                    │
│  ┌─────────────────────────────────────────────────┐   │
│  │  Hook Layer 3: SQLQueryListener                  │   │
│  │  • AfterQuery (log results, metrics)             │   │
│  └─────────────────────────────────────────────────┘   │
│                     ↓                                    │
│  ┌─────────────────────────────────────────────────┐   │
│  │  Hook Layer 4: ResourceNormalizer                │   │
│  │  • NormalizeOne/Many                             │   │
│  │  • Transform response                            │   │
│  └─────────────────────────────────────────────────┘   │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                   HTTP Response                          │
└─────────────────────────────────────────────────────────┘
```

## Quick Start

1. **Create hooks file** for your resource
2. **Implement only needed hooks** (embed NoOpHooks for defaults)
3. **Register hooks** in route registration
4. **Test** with API calls

See `HOOKS_GUIDE.md` for detailed examples and patterns!

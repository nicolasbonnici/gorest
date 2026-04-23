# Processor Module

The Processor module provides a unified API processor that wraps all API layers (response, serializer, hooks, middleware, pagination, filtering) to eliminate handler boilerplate and ensure consistent endpoint behavior.

## Overview

Inspired by API Platform's processor concept, the GoREST Processor module:

- **Eliminates boilerplate** - Handlers become one-liners calling processor methods
- **Standardizes behavior** - Consistent error handling, pagination, filtering across all endpoints
- **Composes existing layers** - Leverages CRUD, hooks, response, serializer, pagination, filter packages
- **Maintains flexibility** - Supports customization through hooks and configuration
- **Backward compatible** - Optional, doesn't break existing code
- **Type-safe** - Uses generics like existing CRUD[T] pattern

## Quick Start

### Basic Usage

```go
package resources

import (
	"example.com/basic-api/generated/dtos"
	"example.com/basic-api/generated/models"
	"example.com/basic-api/hooks"
	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/processor"
)

type TodoResource struct {
	processor processor.Processor[models.Todo, dtos.TodoCreateDTO, dtos.TodoUpdateDTO, dtos.TodoDTO]
}

func NewTodoResource(db database.Database) *TodoResource {
	todoCRUD := crud.NewWithHooks(db, &hooks.TodoHooks{})

	todoProcessor := processor.New(processor.ProcessorConfig[
		models.Todo,
		dtos.TodoCreateDTO,
		dtos.TodoUpdateDTO,
		dtos.TodoDTO,
	]{
		DB:                 db,
		CRUD:               todoCRUD,
		PaginationLimit:    30,
		PaginationMaxLimit: 100,
		AllowedFields:      []string{"id", "user_id", "title", "content", "created_at"},
		Converter: &processor.FuncConverter[models.Todo, dtos.TodoCreateDTO, dtos.TodoUpdateDTO, dtos.TodoDTO]{
			CreateToModel: todoCreateDTOToModel,
			UpdateToModel: todoUpdateDTOToModel,
			ModelToDTO:    modelToTodoDTO,
		},
		ContextEnrichers: []processor.ContextEnricher{
			processor.UserIDEnricher("UserId"),
		},
	})

	return &TodoResource{processor: todoProcessor}
}

// Handlers are one-liners
func (r *TodoResource) Create(c *fiber.Ctx) error  { return r.processor.Create(c) }
func (r *TodoResource) GetByID(c *fiber.Ctx) error { return r.processor.GetByID(c) }
func (r *TodoResource) GetAll(c *fiber.Ctx) error  { return r.processor.GetAll(c) }
func (r *TodoResource) Update(c *fiber.Ctx) error  { return r.processor.Update(c) }
func (r *TodoResource) Delete(c *fiber.Ctx) error  { return r.processor.Delete(c) }

// Converter functions
func modelToTodoDTO(m models.Todo) dtos.TodoDTO {
	return dtos.TodoDTO{
		Id:        m.Id,
		UserId:    m.UserId,
		Title:     m.Title,
		Content:   m.Content,
		CreatedAt: m.CreatedAt,
	}
}

func todoCreateDTOToModel(dto dtos.TodoCreateDTO) models.Todo {
	return models.Todo{
		UserId:  dto.UserId,
		Title:   dto.Title,
		Content: dto.Content,
	}
}

func todoUpdateDTOToModel(dto dtos.TodoUpdateDTO) models.Todo {
	return models.Todo{
		UserId:  dto.UserId,
		Title:   dto.Title,
		Content: dto.Content,
	}
}
```

## Configuration

### ProcessorConfig Fields

#### Required Fields

- **DB** (`database.Database`) - Database connection
- **CRUD** (`*crud.CRUD[TModel]`) - CRUD instance with hooks
- **Converter** (`ModelConverter`) - DTO/Model conversion functions

#### Pagination Settings

- **PaginationLimit** (`int`) - Default pagination limit (default: 30)
- **PaginationMaxLimit** (`int`) - Maximum pagination limit (default: 100)

#### Filtering and Ordering

- **AllowedFields** (`[]string`) - Fields allowed for filtering/ordering
- **FieldMap** (`map[string]string`) - Optional JSON→DB field mapping

Example:
```go
AllowedFields: []string{"id", "title", "created_at"}
// or
FieldMap: map[string]string{
	"id":        "id",
	"userId":    "user_id",
	"title":     "title",
	"createdAt": "created_at",
}
```

#### Context Enrichment

- **ContextEnrichers** (`[]ContextEnricher`) - Functions to enrich models from context

Built-in enrichers:
```go
ContextEnrichers: []processor.ContextEnricher{
	processor.UserIDEnricher("UserId"),      // Auto-populate from gorest-auth
	processor.TenantIDEnricher("TenantId"),  // Auto-populate from context
}
```

#### Error Handling

- **ErrorHandler** (`ErrorHandler`) - Custom error handler (default: `DefaultErrorHandler`)

The default error handler provides:
- Parse errors → 400 Bad Request
- Validation errors → 400 Bad Request
- Invalid ID → 400 Bad Request
- Not found → 404 Not Found
- Database errors → 500 Internal Server Error

#### Validation

- **ValidateCreate** (`func(TCreateDTO) error`) - Optional create validation
- **ValidateUpdate** (`func(TUpdateDTO) error`) - Optional update validation

Example:
```go
ValidateCreate: func(dto dtos.TodoCreateDTO) error {
	if dto.Title == "" {
		return fmt.Errorf("title is required")
	}
	return nil
}
```

## Customization with Hooks

The processor supports custom hooks that run before CRUD operations using a fluent API:

### Create Hook

```go
todoProcessor := processor.New(config).
	WithCreateHook(func(c *fiber.Ctx, dto dtos.TodoCreateDTO, model *models.Todo) error {
		// Custom validation or enrichment before CRUD.Create()
		if dto.Title == "" {
			return fmt.Errorf("title is required")
		}
		model.Slug = generateSlug(dto.Title)
		return nil
	})
```

### Update Hook

```go
todoProcessor := processor.New(config).
	WithUpdateHook(func(c *fiber.Ctx, dto dtos.TodoUpdateDTO, model *models.Todo) error {
		// Custom logic before CRUD.Update()
		model.UpdatedAt = time.Now()
		return nil
	})
```

### Delete Hook

```go
todoProcessor := processor.New(config).
	WithDeleteHook(func(c *fiber.Ctx, id any) error {
		// Custom validation before CRUD.Delete()
		if !canDelete(c, id) {
			return fmt.Errorf("permission denied")
		}
		return nil
	})
```

### GetByID Hook

```go
todoProcessor := processor.New(config).
	WithGetByIDHook(func(c *fiber.Ctx, id any) error {
		// Custom logic before CRUD.GetByID()
		logger.Log.Info("Fetching todo", "id", id)
		return nil
	})
```

### GetAll Hook

```go
todoProcessor := processor.New(config).
	WithGetAllHook(func(c *fiber.Ctx, conditions *[]query.Condition, orderBy *[]crud.OrderByClause) error {
		// Add custom filters (e.g., multi-tenancy)
		tenantID := c.Locals("tenant_id").(string)
		*conditions = append(*conditions, query.Eq("tenant_id", tenantID))
		return nil
	})
```

### Chaining Multiple Hooks

```go
todoProcessor := processor.New(config).
	WithCreateHook(createHook).
	WithUpdateHook(updateHook).
	WithGetAllHook(getAllHook)
```

## Operation Flow

### Create Operation

1. Parse CreateDTO from request body
2. Validate DTO (if ValidateCreate provided)
3. Convert DTO to Model
4. Apply ContextEnrichers (user_id, tenant_id)
5. Run custom CreateHook (if provided)
6. Call CRUD.Create() - all hook layers execute:
   - **Authorization.ValidateWrite()** - Check field-level write permissions
   - **Authorization.CheckCreate()** - Check resource-level create permission
   - **StateProcessor** - Validation, enrichment, business logic
   - **SQLQueryBuilderModifier** - Query modification (if applicable)
   - **SQLQueryListener.BeforeQuery()** - Pre-execution observation
   - Database INSERT execution
   - **SQLQueryListener.AfterQuery()** - Post-execution observation
7. Fetch created entity with GetByID (includes CheckRead, FilterRead)
8. Convert to ResponseDTO
9. Send 201 Created with response.SendFormatted()

### GetByID Operation

1. Extract ID from path params
2. Run custom GetByIDHook (if provided)
3. Call CRUD.GetByID() - all hook layers execute:
   - **SQLQueryBuilderModifier.ModifySelectQuery()** - Add WHERE, JOIN conditions
   - **SQLQueryListener.BeforeQuery()** - Pre-execution observation
   - Database SELECT execution
   - **SQLQueryListener.AfterQuery()** - Post-execution observation
   - **Authorization.CheckRead()** - Check resource-level read permission (returns 404 if denied)
   - **Authorization.FilterRead()** - Remove fields user cannot read
   - **Serializer.SerializeOne()** - Response transformation
4. Convert to ResponseDTO
5. Send 200 OK with response.SendFormatted()

### GetAll Operation

1. Parse pagination params (limit, page, count)
2. Parse filters using filter.FilterSet (supports eq, ne, gt, gte, lt, lte, like, in, nin)
3. Parse ordering using filter.OrderSet
4. Run custom GetAllHook (if provided)
5. Call CRUD.GetAllPaginated() - all hook layers execute:
   - **SQLQueryBuilderModifier.ModifySelectQuery()** - Add WHERE, JOIN conditions (multi-tenancy, etc.)
   - **SQLQueryListener.BeforeQuery()** - Pre-execution observation
   - Database SELECT execution with pagination
   - **SQLQueryListener.AfterQuery()** - Post-execution observation
   - For each item:
     - **Authorization.CheckRead()** - Check read permission (item removed from results if denied)
     - **Authorization.FilterRead()** - Remove forbidden fields
   - **Serializer.SerializeMany()** - Response transformation
6. Convert models to ResponseDTOs
7. Send paginated response using pagination.SendHydraCollection()

### Update Operation

1. Extract ID from path params
2. Parse UpdateDTO from request body
3. Validate DTO (if ValidateUpdate provided)
4. Convert DTO to Model
5. Apply ContextEnrichers
6. Run custom UpdateHook (if provided)
7. Call CRUD.Update() - all hook layers execute:
   - **Authorization.ValidateWrite()** - Check field-level write permissions
   - **Authorization.CheckUpdate()** - Check resource-level update permission
   - **StateProcessor** - Validation, enrichment, business logic
   - **SQLQueryBuilderModifier.ModifyUpdateQuery()** - Query modification (tenant isolation, etc.)
   - **SQLQueryListener.BeforeQuery()** - Pre-execution observation
   - Database UPDATE execution
   - **SQLQueryListener.AfterQuery()** - Post-execution observation
8. Convert to ResponseDTO
9. Send 200 OK with response.SendFormatted()

### Delete Operation

1. Extract ID from path params
2. Run custom DeleteHook (if provided)
3. Call CRUD.Delete() - all hook layers execute:
   - **Authorization.CheckDelete()** - Check resource-level delete permission
   - **StateProcessor** - Validation, business logic (soft delete implementation)
   - **SQLQueryBuilderModifier.ModifyDeleteQuery()** - Query modification (soft delete, cascades)
   - **SQLQueryListener.BeforeQuery()** - Pre-execution observation
   - Database DELETE execution
   - **SQLQueryListener.AfterQuery()** - Post-execution observation
4. Send 204 No Content

## Integration with Existing Layers

### CRUD Layer
- Processor calls CRUD.Create(), CRUD.GetByID(), etc.
- All hook layers execute as before (Authorization, StateProcessor, ModifyQuery, BeforeQuery, AfterQuery, Serializer)
- No changes needed to crud/ package

### Response Layer
- Uses response.SendFormatted() for single items
- Uses pagination.SendHydraCollection() for lists
- Maintains JSON vs JSON-LD content negotiation
- Supports expand[] query parameter for relation expansion

### Filter/Pagination Layer
- Parses filters using filter.FilterSet
- Parses ordering using filter.OrderSet
- Converts to crud.PaginationOptions

### Hooks Layer
- All **5 hook layers** execute during CRUD operations (in order):
  1. **Authorization** (Layer 5) - RBAC checks (CheckCreate/Read/Update/Delete, ValidateWrite, FilterRead)
  2. **StateProcessor** (Layer 1) - Validation, enrichment, business logic
  3. **SQLQueryBuilderModifier** (Layer 3) - Query modification (WHERE, JOIN, etc.)
  4. **SQLQueryListener** (Layer 2) - BeforeQuery/AfterQuery observation
  5. **Serializer** (Layer 4) - Response transformation
- Processor hooks (WithCreateHook, etc.) run BEFORE CRUD operations

### Authorization Layer (RBAC)
- **Mandatory** - All resources must implement `Authorization[T]` interface
- Field-level permissions via `rbac:` struct tags (read/write roles)
- Resource-level permissions via hook methods (CheckCreate/Read/Update/Delete)
- Automatic field filtering with `FilterRead()` removes forbidden fields
- Write validation with `ValidateWrite()` blocks unauthorized field changes
- Voter system resolves role hierarchies and checks permissions
- See [AUTHORIZATION.md](../AUTHORIZATION.md) and [RBAC.md](../RBAC.md) for details

### Auth Plugin
- ContextEnrichers extract user_id from auth.GetAuthenticatedUser()
- Roles stored in context via `rbac.WithRoles(ctx, roles)` by auth middleware
- Authorization layer uses `rbac.GetRoles(ctx)` to check permissions

## Advanced Examples

### Multi-Tenancy

```go
todoProcessor := processor.New(config).
	WithGetAllHook(func(c *fiber.Ctx, conditions *[]query.Condition, orderBy *[]crud.OrderByClause) error {
		tenantID := c.Locals("tenant_id").(string)
		*conditions = append(*conditions, query.Eq("tenant_id", tenantID))
		return nil
	}).
	WithGetByIDHook(func(c *fiber.Ctx, id any) error {
		// Tenant filtering happens in hooks layer via ModifySelectQuery
		return nil
	})
```

### Role-Based Access Control (RBAC)

Configure RBAC for your resources using the Authorization layer:

```go
import (
	"github.com/nicolasbonnici/gorest/hooks"
	"github.com/nicolasbonnici/gorest/rbac"
)

// Define model with rbac tags
type Article struct {
	ID        string    `json:"id" db:"id"`
	Title     string    `json:"title" db:"title"`
	Content   string    `json:"content" db:"content" rbac:"read:editor,admin write:admin"`
	Draft     bool      `json:"draft" db:"draft" rbac:"read:author,editor,admin write:author,editor,admin"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	AuthorID  string    `json:"author_id" db:"author_id"`
}

// Custom hooks with Authorization
type ArticleHooks struct {
	*hooks.DefaultAuthorization[models.Article]
	hooks.NoOpHooks[models.Article]
}

func NewArticleHooks(rbacConfig rbac.Config) *ArticleHooks {
	return &ArticleHooks{
		DefaultAuthorization: hooks.NewDefaultAuthorization[models.Article](rbacConfig),
	}
}

// Override CheckRead for ownership-based access
func (h *ArticleHooks) CheckRead(ctx context.Context, article *models.Article) error {
	userID, _ := rbac.GetUserID(ctx)
	roles, _ := rbac.GetRoles(ctx)

	// Admins can read all
	if h.GetVoter().IsSuperuser(roles) {
		return nil
	}

	// Authors can read their own articles
	if article.AuthorID == userID {
		return nil
	}

	// Others need editor or admin role for published articles
	if !article.Draft && rbac.HasAnyRole(roles, []string{"editor", "admin"}, h.GetVoter().GetConfig().RoleHierarchy) {
		return nil
	}

	return rbac.ErrNotFound // 404 for security
}

// Create processor with RBAC-enabled hooks
articleCRUD := crud.NewWithHooks(db, NewArticleHooks(rbacConfig))
articleProcessor := processor.New(processor.ProcessorConfig[...]{
	DB:   db,
	CRUD: articleCRUD,
	// ... other config
})
```

**Field-level permissions:**
- `Content` field: Readable by editors/admins, writable by admins only
- `Draft` field: Readable/writable by authors, editors, admins
- Fields without `rbac:` tags inherit from `DefaultFieldPolicy` config

**Resource-level permissions:**
- Ownership checks in `CheckRead/CheckUpdate/CheckDelete`
- Role-based access in `CheckCreate`
- Automatic field filtering via `FilterRead()`
- Write validation via `ValidateWrite()`

See [RBAC.md](../RBAC.md) for complete tag syntax and configuration options.

### Custom Validation

```go
config := processor.ProcessorConfig[...]{
	// ... other config
	ValidateCreate: func(dto dtos.TodoCreateDTO) error {
		if dto.Title == "" {
			return fmt.Errorf("title is required")
		}
		if len(dto.Title) > 100 {
			return fmt.Errorf("title must be less than 100 characters")
		}
		return nil
	},
}
```

### Custom Error Handling

```go
type CustomErrorHandler struct{}

func (h *CustomErrorHandler) HandleError(c *fiber.Ctx, err error, operation string) error {
	// Log error
	logger.Log.Error("Processor error", "operation", operation, "error", err)

	// Custom error responses
	if operation == "validate" {
		return c.Status(422).JSON(fiber.Map{
			"error": "Validation failed",
			"details": err.Error(),
		})
	}

	// Fallback to default
	return (&processor.DefaultErrorHandler{}).HandleError(c, err, operation)
}

config := processor.ProcessorConfig[...]{
	// ... other config
	ErrorHandler: &CustomErrorHandler{},
}
```

## Migration Guide

### From Manual Handlers to Processor

**Before:**
```go
func (r *TodoResource) Create(c *fiber.Ctx) error {
	var createDTO dtos.TodoCreateDTO
	if err := c.BodyParser(&createDTO); err != nil {
		logger.Log.Error("Failed to parse request body", "error", err)
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	item := todoCreateDTOToModel(createDTO)

	if user := auth.GetAuthenticatedUser(c); user != nil {
		item.UserId = &user.UserID
	}

	ctx := auth.Context(c)
	if err := r.CRUD.Create(ctx, item); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	created, err := r.CRUD.GetByID(ctx, item.Id)
	if err != nil {
		dto := modelToTodoDTO(item)
		return response.SendFormatted(c, 201, dto)
	}

	dto := modelToTodoDTO(*created)
	return response.SendFormatted(c, 201, dto)
}
```

**After:**
```go
func (r *TodoResource) Create(c *fiber.Ctx) error {
	return r.processor.Create(c)
}
```

### Benefits

- **90% boilerplate reduction** for handlers
- **Consistent error handling** across all endpoints
- **Built-in pagination/filtering/ordering** - no repetition
- **Standard auth integration** via ContextEnrichers
- **Easy to test** - mock converters, error handlers, enrichers
- **Clear control flow** - composition, not magic

## Best Practices

1. **Use FieldMap for consistency** - Map JSON field names to DB columns for filtering/ordering
2. **Apply ContextEnrichers for common fields** - user_id, tenant_id, organization_id
3. **Use custom hooks sparingly** - Prefer using the existing hooks layer (StateProcessor, etc.)
4. **Validate at the DTO layer** - Use ValidateCreate/ValidateUpdate for business rules
5. **Keep converters simple** - Just map fields, don't add logic
6. **Test converters independently** - They're pure functions, easy to unit test
7. **Use custom error handlers for domain-specific errors** - Extend DefaultErrorHandler

## Backward Compatibility

- 100% backward compatible - processor is completely optional
- Existing handlers continue working without changes
- Can migrate resource by resource
- No breaking changes to any existing packages
- Drop-in replacement for generated resources (update gorest-codegen templates)

## Testing

The processor module is designed to be testable:

```go
func TestCreate(t *testing.T) {
	// Create test database
	db := testhelpers.SetupPostgres(t)
	defer db.Close()

	// Create processor with test config
	todoCRUD := crud.New[models.Todo](db)
	todoProcessor := processor.New(processor.ProcessorConfig[...]{
		DB:   db,
		CRUD: todoCRUD,
		Converter: &processor.FuncConverter[...]{
			CreateToModel: todoCreateDTOToModel,
			UpdateToModel: todoUpdateDTOToModel,
			ModelToDTO:    modelToTodoDTO,
		},
	})

	// Create test Fiber app and context
	app := fiber.New()
	// ... test the processor methods
}
```

## Summary

The Processor module is a powerful abstraction that eliminates boilerplate while maintaining the flexibility of GoREST's hook system. It's:

- **Optional** - Adopt incrementally
- **Composable** - Works with all existing layers
- **Customizable** - Hooks for special cases
- **Type-safe** - Full generic support
- **Testable** - Easy to mock and test

For most resources, it reduces handlers to simple one-liners while providing consistent behavior across your entire API.

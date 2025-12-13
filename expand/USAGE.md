# Expand Query Parameter Usage

The expand functionality allows deserializing IRIs into full nested objects for both JSON and JSON-LD responses.

## Query Syntax

```bash
# Single relation
GET /todos?expand[]=user

# Multiple relations
GET /todos?expand[]=user&expand[]=comments

# Combined with other parameters
GET /todos?limit=10&order[created_at]=desc&expand[]=user
```

## Implementation in Resource Handlers

### 1. Define Relation Configurations

```go
import (
	"github.com/nicolasbonnici/gorest/expand"
	"github.com/nicolasbonnici/gorest/crud"
)

// In your resource struct, add CRUD instances for related tables
type TodoResource struct {
	DB                 database.Database
	CRUD               *crud.CRUD[models.Todo]
	UserCRUD           *crud.CRUD[models.User]
	PaginationLimit    int
	PaginationMaxLimit int
}

// Define expand configurations
func (r *TodoResource) getExpandConfigs() map[string]expand.RelationConfig {
	return map[string]expand.RelationConfig{
		"user": {
			Field:           "user",
			ForeignKeyField: "userId",
			RelatedTable:    "users",
			CRUD:            r.UserCRUD,
		},
	}
}
```

### 2. Modify List Endpoint

```go
func (r *TodoResource) List(c *fiber.Ctx) error {
	// ... existing pagination and filter code ...

	result, err := r.CRUD.GetAllPaginated(ctx, crud.PaginationOptions{
		Limit:         limit,
		Offset:        offset,
		IncludeCount:  includeCount,
		WhereClause:   whereClause,
		WhereArgs:     whereArgs,
		OrderByClause: orderByClause,
	})
	if err != nil {
		return pagination.SendPaginatedError(c, 500, err.Error())
	}

	// Convert to DTOs
	dtoItems := make([]dtos.TodoDTO, len(result.Items))
	for i, item := range result.Items {
		dtoItems[i] = modelToTodoDTO(item)
	}

	// Expand relations if requested
	expandParams := response.ParseExpandQuery(c)
	if len(expandParams) > 0 {
		configs := r.getExpandConfigs()
		validExpand := expand.ParseExpand(expandParams, configs)

		expandedData, err := expand.ExpandRelations(ctx, dtoItems, validExpand, configs)
		if err == nil {
			dtoItems = expandedData.([]interface{})
		}
	}

	return pagination.SendHydraCollection(c, dtoItems, result.Total, limit, page, r.PaginationLimit)
}
```

### 3. Modify Get Endpoint

```go
func (r *TodoResource) Get(c *fiber.Ctx) error {
	ctx := auth.ExtractContext(c)
	id := c.Params("id")

	item, err := r.CRUD.GetByID(ctx, id)
	if err != nil {
		return response.SendError(c, 404, "Not found")
	}

	dto := modelToTodoDTO(*item)

	// Expand relations if requested
	expandParams := response.ParseExpandQuery(c)
	if len(expandParams) > 0 {
		configs := r.getExpandConfigs()
		validExpand := expand.ParseExpand(expandParams, configs)

		expandedData, err := expand.ExpandRelations(ctx, dto, validExpand, configs)
		if err == nil {
			return response.SendFormatted(c, 200, expandedData)
		}
	}

	return response.SendFormatted(c, 200, dto)
}
```

## Response Examples

### Without Expand (Default)

```bash
GET /todos/todo-123
```

```json
{
  "@context": "https://schema.org/",
  "@type": "TodoDTO",
  "@id": "/todos/todo-123",
  "id": "todo-123",
  "userId": "/users/user-456",
  "title": "Buy groceries",
  "content": "Milk, eggs, bread"
}
```

### With Expand

```bash
GET /todos/todo-123?expand[]=user
```

```json
{
  "@context": "https://schema.org/",
  "@type": "TodoDTO",
  "@id": "/todos/todo-123",
  "id": "todo-123",
  "user": {
    "@type": "UserDTO",
    "@id": "/users/user-456",
    "id": "user-456",
    "name": "Alice",
    "email": "alice@example.com"
  },
  "title": "Buy groceries",
  "content": "Milk, eggs, bread"
}
```

## Features

- ✅ Works with both JSON and JSON-LD serializers
- ✅ Supports collections and single items
- ✅ Validates expand parameters against configured relations
- ✅ Gracefully handles missing or null foreign keys
- ✅ One-level expansion (no nested like `user.profile`)
- ✅ Respects DTO field visibility rules
- ✅ Case-insensitive field matching

## Technical Implementation

1. **Query Parsing**: `response.ParseExpandQuery(c)` extracts `expand[]` parameters
2. **Validation**: `expand.ParseExpand()` validates against allowed relations
3. **Fetching**: `expand.ExpandRelations()` fetches related objects via CRUD
4. **Serialization**: Serializer replaces IRIs with nested objects
5. **Response**: Normal `SendFormatted()` or `SendHydraCollection()`

## Notes

- Expand is optional - works transparently with existing endpoints
- No database schema changes required
- No model changes required (uses map[string]interface{} internally)
- Fetches relations separately (not SQL JOINs)
- DTOs for expanded objects use their own DTO rules

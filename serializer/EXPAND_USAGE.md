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
	"github.com/nicolasbonnici/gorest/serializer"
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
func (r *TodoResource) getExpandConfigs() map[string]serializer.RelationConfig {
	return map[string]serializer.RelationConfig{
		"user": {
			Field:           "user",
			ForeignKeyField: "userId",
			RelatedTable:    "users",
			Fetcher:         crud.RelationFetcher(r.UserCRUD),
		},
	}
}
```

`Fetcher` is a `serializer.RelationFetcher`. `crud.RelationFetcher` adapts any
`*crud.CRUD[T]` to it, resolving a whole page with **one query per relation**
instead of one per item: expanding 25 todos over 2 relations costs 2 queries, not
50. Batched lookups go through `CRUD.GetByIDs`, so `ModifySelectQuery` scoping
(multi-tenancy, soft deletes) and the read authorization hooks apply exactly as
they would on a single `GetByID`.

Implement `RelationFetcher` yourself to source a relation from somewhere other
than a CRUD instance (a cache, an upstream service):

```go
type RelationFetcher interface {
	FetchByIDs(ctx context.Context, ids []string) (map[string]any, error)
}
```

The returned map is keyed by identifier. Missing keys are not an error: the item
keeps its foreign key unexpanded. A fetcher returning an error leaves that one
relation unexpanded rather than failing the whole response.

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
		validExpand := serializer.ParseExpand(expandParams, configs)

		expandedData, err := serializer.ExpandRelations(ctx, dtoItems, validExpand, configs)
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
		validExpand := serializer.ParseExpand(expandParams, configs)

		expandedData, err := serializer.ExpandRelations(ctx, dto, validExpand, configs)
		if err == nil {
			return response.SendFormatted(c, 200, expandedData)
		}
	}

	return response.SendFormatted(c, 200, dto)
}
```

## Response Examples

### Without Expand (Default - IRI)

```bash
GET /todos/todo-123
```

```json
{
  "@context": "https://schema.org/",
  "@type": "TodoDTO",
  "@id": "/todos/todo-123",
  "id": "todo-123",
  "user": "/users/user-456",
  "title": "Buy groceries",
  "content": "Milk, eggs, bread"
}
```

**Note:** Foreign keys are automatically converted to clean relation names (e.g., `user` instead of `userId`) with IRI values.

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

**Note:** Foreign key fields (like `userId`) are automatically converted to clean relation names (`user`):
- Without expand: `"user": "/users/user-456"` (IRI)
- With expand: `"user": { "id": "user-456", ... }` (full object)

## Features

- ✅ Works with both JSON and JSON-LD serializers
- ✅ Supports collections and single items
- ✅ Validates expand parameters against configured relations
- ✅ Gracefully handles missing or null foreign keys
- ✅ One-level expansion (no nested like `user.profile`)
- ✅ Respects DTO field visibility rules
- ✅ Case-insensitive field matching
- ✅ Clean API: Relation names (`user`) instead of FK fields (`userId`) everywhere
- ✅ Consistent field names whether expanded or not

## Technical Implementation

1. **Query Parsing**: `response.ParseExpandQuery(c)` extracts `expand[]` parameters
2. **Validation**: `serializer.ParseExpand()` validates against allowed relations
3. **Fetching**: `serializer.ExpandRelations()` fetches related objects via CRUD
4. **Serialization**: Serializer replaces IRIs with nested objects
5. **Response**: Normal `SendFormatted()` or `SendHydraCollection()`

## Notes

- **Clean relation names**: Foreign key fields (e.g., `userId`) are automatically renamed to relation names (e.g., `user`) in all responses
- **Consistent naming**: Whether you use expand or not, you always get `user` (not `userId`) in your API responses
- **IRIs by default**: Without expand, relations are IRIs like `"/users/123"`
- **Full objects with expand**: With expand, relations become nested objects
- Expand is optional - works transparently with existing endpoints
- No database schema changes required
- No model changes required (uses map[string]interface{} internally)
- Fetches relations separately (not SQL JOINs), one batched query per relation
- DTOs for expanded objects use their own DTO rules

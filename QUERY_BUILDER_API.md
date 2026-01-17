# GoREST Query Builder API Proposal

## Overview

A fluent, chainable query builder for GoREST that simplifies SQL query construction while maintaining database abstraction and type safety.

## Design Goals

1. **Fluent Interface**: Chainable methods for intuitive query construction
2. **Database Agnostic**: Works with PostgreSQL, MySQL, and SQLite via Dialect
3. **Type Safe**: Leverages Go's type system where possible
4. **Zero String Concatenation**: No manual SQL string building
5. **Integration**: Seamless integration with existing CRUD hooks
6. **Performance**: Minimal overhead, efficient query generation

## Core API

### Package Structure

```
gorest/
└── query/
    ├── builder.go       # Main query builder
    ├── select.go        # SELECT query builder
    ├── insert.go        # INSERT query builder
    ├── update.go        # UPDATE query builder
    ├── delete.go        # DELETE query builder
    ├── condition.go     # WHERE conditions
    ├── join.go          # JOIN clauses
    └── expression.go    # SQL expressions
```

### 1. Query Builder Factory

```go
package query

import (
    "github.com/nicolasbonnici/gorest/database"
)

// Builder creates queries for a specific dialect
type Builder struct {
    dialect database.Dialect
}

// New creates a new query builder
func New(dialect database.Dialect) *Builder {
    return &Builder{dialect: dialect}
}

// Select starts a SELECT query
func (b *Builder) Select(columns ...string) *SelectBuilder

// Insert starts an INSERT query
func (b *Builder) Insert(table string) *InsertBuilder

// Update starts an UPDATE query
func (b *Builder) Update(table string) *UpdateBuilder

// Delete starts a DELETE query
func (b *Builder) Delete(table string) *DeleteBuilder
```

### 2. SELECT Query Builder

```go
type SelectBuilder struct {
    dialect     database.Dialect
    columns     []string
    table       string
    joins       []joinClause
    conditions  []Condition
    groupBy     []string
    having      []Condition
    orderBy     []orderClause
    limit       int
    offset      int
    distinct    bool
}

// Example Usage:
query.New(dialect).
    Select("id", "name", "email").
    From("users").
    Where(Eq("status", "active")).
    Where(Gt("age", 18)).
    OrderBy("created_at", DESC).
    Limit(10).
    Offset(20).
    Build()

// Methods:
func (s *SelectBuilder) From(table string) *SelectBuilder
func (s *SelectBuilder) Distinct() *SelectBuilder
func (s *SelectBuilder) Where(condition Condition) *SelectBuilder
func (s *SelectBuilder) And(condition Condition) *SelectBuilder
func (s *SelectBuilder) Or(condition Condition) *SelectBuilder
func (s *SelectBuilder) Join(table string, on Condition) *SelectBuilder
func (s *SelectBuilder) LeftJoin(table string, on Condition) *SelectBuilder
func (s *SelectBuilder) RightJoin(table string, on Condition) *SelectBuilder
func (s *SelectBuilder) InnerJoin(table string, on Condition) *SelectBuilder
func (s *SelectBuilder) GroupBy(columns ...string) *SelectBuilder
func (s *SelectBuilder) Having(condition Condition) *SelectBuilder
func (s *SelectBuilder) OrderBy(column string, direction Order) *SelectBuilder
func (s *SelectBuilder) Limit(limit int) *SelectBuilder
func (s *SelectBuilder) Offset(offset int) *SelectBuilder
func (s *SelectBuilder) Build() (query string, args []any)
```

### 3. INSERT Query Builder

```go
type InsertBuilder struct {
    dialect database.Dialect
    table   string
    columns []string
    values  [][]any
    returning []string
}

// Example Usage:
query.New(dialect).
    Insert("users").
    Columns("name", "email", "age").
    Values("John Doe", "john@example.com", 30).
    Values("Jane Doe", "jane@example.com", 25).
    Returning("id").
    Build()

// Methods:
func (i *InsertBuilder) Columns(columns ...string) *InsertBuilder
func (i *InsertBuilder) Values(values ...any) *InsertBuilder
func (i *InsertBuilder) ValuesMap(values map[string]any) *InsertBuilder
func (i *InsertBuilder) Returning(columns ...string) *InsertBuilder
func (i *InsertBuilder) Build() (query string, args []any)
```

### 4. UPDATE Query Builder

```go
type UpdateBuilder struct {
    dialect    database.Dialect
    table      string
    sets       map[string]any
    conditions []Condition
    returning  []string
}

// Example Usage:
query.New(dialect).
    Update("users").
    Set("name", "John Updated").
    Set("email", "john.new@example.com").
    Where(Eq("id", 123)).
    Returning("updated_at").
    Build()

// Methods:
func (u *UpdateBuilder) Set(column string, value any) *UpdateBuilder
func (u *UpdateBuilder) SetMap(values map[string]any) *UpdateBuilder
func (u *UpdateBuilder) Where(condition Condition) *UpdateBuilder
func (u *UpdateBuilder) And(condition Condition) *UpdateBuilder
func (u *UpdateBuilder) Or(condition Condition) *UpdateBuilder
func (u *UpdateBuilder) Returning(columns ...string) *UpdateBuilder
func (u *UpdateBuilder) Build() (query string, args []any)
```

### 5. DELETE Query Builder

```go
type DeleteBuilder struct {
    dialect    database.Dialect
    table      string
    conditions []Condition
    returning  []string
}

// Example Usage:
query.New(dialect).
    Delete("users").
    Where(Eq("status", "inactive")).
    Where(Lt("last_login", time.Now().AddDate(0, -6, 0))).
    Build()

// Methods:
func (d *DeleteBuilder) Where(condition Condition) *DeleteBuilder
func (d *DeleteBuilder) And(condition Condition) *DeleteBuilder
func (d *DeleteBuilder) Or(condition Condition) *DeleteBuilder
func (d *DeleteBuilder) Returning(columns ...string) *DeleteBuilder
func (d *DeleteBuilder) Build() (query string, args []any)
```

### 6. Conditions API

```go
type Condition interface {
    ToSQL(dialect database.Dialect, paramStart int) (sql string, args []any)
}

// Comparison Operators
func Eq(column string, value any) Condition      // column = ?
func Ne(column string, value any) Condition      // column != ?
func Gt(column string, value any) Condition      // column > ?
func Gte(column string, value any) Condition     // column >= ?
func Lt(column string, value any) Condition      // column < ?
func Lte(column string, value any) Condition     // column <= ?

// Pattern Matching
func Like(column string, pattern string) Condition       // column LIKE ?
func ILike(column string, pattern string) Condition      // column ILIKE ? (case-insensitive)
func NotLike(column string, pattern string) Condition    // column NOT LIKE ?

// Null Checks
func IsNull(column string) Condition                     // column IS NULL
func IsNotNull(column string) Condition                  // column IS NOT NULL

// Range/Set Operators
func In(column string, values ...any) Condition          // column IN (?, ?, ?)
func NotIn(column string, values ...any) Condition       // column NOT IN (?, ?, ?)
func Between(column string, start, end any) Condition    // column BETWEEN ? AND ?

// Logical Operators
func And(conditions ...Condition) Condition              // (cond1 AND cond2 AND ...)
func Or(conditions ...Condition) Condition               // (cond1 OR cond2 OR ...)
func Not(condition Condition) Condition                  // NOT (condition)

// Raw SQL (escape hatch)
func Raw(sql string, args ...any) Condition              // raw SQL fragment

// Column Comparison
func ColEq(col1, col2 string) Condition                  // col1 = col2
func ColNe(col1, col2 string) Condition                  // col1 != col2
func ColGt(col1, col2 string) Condition                  // col1 > col2
func ColLt(col1, col2 string) Condition                  // col1 < col2

// Example Usage:
Where(
    And(
        Eq("status", "active"),
        Or(
            Gt("age", 18),
            Eq("verified", true),
        ),
        In("role", "admin", "moderator"),
        IsNotNull("email"),
    ),
)
```

### 7. Integration with CRUD Hooks

```go
// In hooks package
type SQLQueryBuilder interface {
    BuildQuery(ctx context.Context, operation Operation, builder *query.Builder) (*query.SelectBuilder | *query.UpdateBuilder | *query.DeleteBuilder, bool)
}

// Example implementation:
type CustomQueryHook struct{}

func (h *CustomQueryHook) OverrideQuery(ctx context.Context, op hooks.Operation, id any, model any) (string, []any, bool) {
    // New approach: use query builder instead of manual SQL
    builder := query.New(dialect)

    selectBuilder := builder.
        Select("*").
        From("users").
        Where(query.Eq("id", id)).
        Where(query.Eq("tenant_id", getTenantID(ctx)))

    sql, args := selectBuilder.Build()
    return sql, args, true
}
```

## Advanced Features

### 8. Subqueries

```go
// Subquery in WHERE clause
subquery := query.New(dialect).
    Select("id").
    From("banned_users")

query.New(dialect).
    Select("*").
    From("posts").
    Where(NotIn("user_id", subquery)).
    Build()
// SELECT * FROM posts WHERE user_id NOT IN (SELECT id FROM banned_users)

// Subquery in FROM clause
query.New(dialect).
    Select("avg_score").
    FromSubquery(
        query.New(dialect).
            Select("AVG(score) as avg_score").
            From("reviews"),
        "subquery",
    ).
    Build()
```

### 9. Aggregations

```go
// Aggregate functions
func Count(column string) Expression
func Sum(column string) Expression
func Avg(column string) Expression
func Min(column string) Expression
func Max(column string) Expression

// Example:
query.New(dialect).
    Select(Count("*").As("total"), Avg("score").As("avg_score")).
    From("reviews").
    GroupBy("product_id").
    Having(Gt("COUNT(*)", 10)).
    Build()
// SELECT COUNT(*) as total, AVG(score) as avg_score FROM reviews GROUP BY product_id HAVING COUNT(*) > 10
```

### 10. Case Expressions

```go
func Case() *CaseBuilder

// Example:
query.New(dialect).
    Select(
        "name",
        Case().
            When(Gt("age", 65), "senior").
            When(Gt("age", 18), "adult").
            Else("minor").
            As("age_group"),
    ).
    From("users").
    Build()
```

## Usage Examples

### Example 1: Simple SELECT with WHERE

```go
func GetActiveUsers(db database.Database, minAge int) ([]User, error) {
    builder := query.New(db.Dialect())

    sql, args := builder.
        Select("id", "name", "email", "age").
        From("users").
        Where(query.Eq("status", "active")).
        Where(query.Gte("age", minAge)).
        OrderBy("created_at", query.DESC).
        Limit(100).
        Build()

    rows, err := db.Query(ctx, sql, args...)
    // ... scan rows
}
```

### Example 2: Complex JOIN Query

```go
func GetUserPosts(db database.Database, userID int) ([]Post, error) {
    builder := query.New(db.Dialect())

    sql, args := builder.
        Select("p.id", "p.title", "p.content", "u.name as author").
        From("posts p").
        InnerJoin("users u", query.ColEq("p.user_id", "u.id")).
        LeftJoin("comments c", query.ColEq("p.id", "c.post_id")).
        Where(query.Eq("p.user_id", userID)).
        Where(query.Eq("p.status", "published")).
        GroupBy("p.id", "p.title", "p.content", "u.name").
        Having(query.Gt("COUNT(c.id)", 0)).
        OrderBy("p.created_at", query.DESC).
        Build()

    rows, err := db.Query(ctx, sql, args...)
    // ... scan rows
}
```

### Example 3: Batch INSERT

```go
func CreateUsers(db database.Database, users []User) error {
    builder := query.New(db.Dialect())

    insert := builder.
        Insert("users").
        Columns("name", "email", "age")

    for _, user := range users {
        insert.Values(user.Name, user.Email, user.Age)
    }

    sql, args := insert.Returning("id").Build()

    _, err := db.Exec(ctx, sql, args...)
    return err
}
```

### Example 4: UPDATE with Complex Conditions

```go
func ArchiveOldPosts(db database.Database) error {
    builder := query.New(db.Dialect())

    sixMonthsAgo := time.Now().AddDate(0, -6, 0)

    sql, args := builder.
        Update("posts").
        Set("status", "archived").
        Set("archived_at", time.Now()).
        Where(
            query.And(
                query.Lt("last_updated", sixMonthsAgo),
                query.Eq("status", "published"),
                query.Lt("view_count", 100),
            ),
        ).
        Build()

    _, err := db.Exec(ctx, sql, args...)
    return err
}
```

### Example 5: Integration with CRUD Hooks

```go
type TenantAwareHook struct {
    hooks.NoOpHooks[User]
}

func (h *TenantAwareHook) OverrideQuery(ctx context.Context, op hooks.Operation, id any, model *User) (string, []any, bool) {
    tenantID := getTenantIDFromContext(ctx)
    if tenantID == "" {
        return "", nil, false
    }

    builder := query.New(getDialectFromContext(ctx))

    switch op {
    case hooks.OperationGetByID:
        sql, args := builder.
            Select("*").
            From("users").
            Where(query.Eq("id", id)).
            Where(query.Eq("tenant_id", tenantID)).
            Build()
        return sql, args, true

    case hooks.OperationGetAll:
        sql, args := builder.
            Select("*").
            From("users").
            Where(query.Eq("tenant_id", tenantID)).
            Build()
        return sql, args, true
    }

    return "", nil, false
}
```

## Implementation Phases

### Phase 1: Core (MVP)
- Basic SELECT, INSERT, UPDATE, DELETE builders
- Simple WHERE conditions (Eq, Ne, Gt, Lt, etc.)
- Integration with Dialect interface
- Tests for PostgreSQL, MySQL, SQLite

### Phase 2: Advanced Queries
- JOIN support (INNER, LEFT, RIGHT, FULL)
- Subqueries
- Aggregations (COUNT, SUM, AVG, etc.)
- GROUP BY and HAVING

### Phase 3: Complex Features
- CASE expressions
- Window functions
- CTEs (Common Table Expressions)
- UNION, INTERSECT, EXCEPT

### Phase 4: Developer Experience
- Query debugging (SQL pretty-printing)
- Query explain integration
- Performance hints
- Migration from raw SQL helper

## Benefits

1. **Type Safety**: Catch errors at compile time instead of runtime
2. **Database Agnostic**: Write once, run on any supported database
3. **Maintainability**: Easier to read and modify complex queries
4. **Testability**: Mock builders for unit tests
5. **Security**: Automatic parameterization prevents SQL injection
6. **DRY**: Reusable query components
7. **IDE Support**: Auto-completion for query methods

## Backward Compatibility

- Query builder is optional, existing raw SQL queries continue to work
- Hooks can use either raw SQL or query builder
- Gradual migration path: start with simple queries, move complex ones over time
- Zero breaking changes to existing API

## Questions for Discussion

1. Should we support table aliases in a more type-safe way?
2. Do we need a separate package or integrate into existing `database` package?
3. Should we provide query builder utilities in CRUD layer directly?
4. How should we handle database-specific features (e.g., PostgreSQL arrays)?
5. Should we include a query cache/optimizer?

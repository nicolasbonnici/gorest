# Query Builder

GoREST includes a powerful, type-safe query builder that eliminates manual SQL string construction while maintaining database abstraction across PostgreSQL, MySQL, and SQLite.

## Quick Start

```go
import (
    "github.com/nicolasbonnici/gorest/database"
    "github.com/nicolasbonnici/gorest/query"
)

// Create a query builder with your database dialect
builder := query.New(db.Dialect())

// Build a SELECT query
sql, args := builder.
    Select("id", "name", "email").
    From("users").
    Where(query.Eq("status", "active")).
    OrderBy("created_at", query.DESC).
    Limit(10).
    Build()

// Execute the query
rows, err := db.Query(ctx, sql, args...)
```

## SELECT Queries

### Basic SELECT

```go
// SELECT * FROM users WHERE age > 18 ORDER BY name LIMIT 10
sql, args := query.New(dialect).
    Select("*").
    From("users").
    Where(query.Gt("age", 18)).
    OrderBy("name", query.ASC).
    Limit(10).
    Build()
```

### Multiple Conditions

```go
// SELECT * FROM posts WHERE status = 'published' AND user_id = 123
sql, args := query.New(dialect).
    Select("*").
    From("posts").
    Where(query.Eq("status", "published")).
    Where(query.Eq("user_id", 123)). // Multiple Where() calls are AND'ed
    Build()

// OR conditions
sql, args := query.New(dialect).
    Select("*").
    From("users").
    Where(query.Or(
        query.Eq("role", "admin"),
        query.Eq("role", "moderator"),
    )).
    Build()
```

### JOINs

```go
// SELECT p.*, u.name as author
// FROM posts p
// INNER JOIN users u ON p.user_id = u.id
sql, args := query.New(dialect).
    Select("p.*", "u.name").
    From("posts").
    As("p").
    Join("users", query.ColEq("p.user_id", "u.id")).
    JoinAs("users", "u", query.ColEq("p.user_id", "u.id")).
    Build()

// LEFT JOIN
builder.LeftJoin("comments", query.ColEq("p.id", "c.post_id"))

// Multiple joins
sql, args := query.New(dialect).
    Select("o.id", "u.name", "p.title").
    From("orders").
    As("o").
    Join("users", query.ColEq("o.user_id", "u.id")).
    JoinAs("users", "u", query.ColEq("o.user_id", "u.id")).
    Join("products", query.ColEq("o.product_id", "p.id")).
    JoinAs("products", "p", query.ColEq("o.product_id", "p.id")).
    Build()
```

### Aggregations

```go
// SELECT COUNT(*) as total, AVG(price) as avg_price
// FROM products
// GROUP BY category
// HAVING COUNT(*) > 10
sql, args := query.New(dialect).
    SelectExpr(
        query.Count(query.Col("*")).As("total"),
        query.Avg(query.Col("price")).As("avg_price"),
    ).
    From("products").
    GroupBy("category").
    HavingExpr(query.Gt(query.Count(query.Col("*")), 10)).
    Build()
```

### Subqueries

```go
// SELECT * FROM users
// WHERE id IN (SELECT user_id FROM banned_users)
subquery := query.New(dialect).
    Select("user_id").
    From("banned_users")

sql, args := query.New(dialect).
    Select("*").
    From("users").
    Where(query.InSubquery("id", subquery)).
    Build()
```

## INSERT Queries

### Single Row

```go
// INSERT INTO users (name, email) VALUES ('John', 'john@example.com')
sql, args := query.New(dialect).
    Insert("users").
    Columns("name", "email").
    Values("John Doe", "john@example.com").
    Build()
```

### Multiple Rows (Batch)

```go
// INSERT INTO users (name, email) VALUES ('John', 'john@...'), ('Jane', 'jane@...')
sql, args := query.New(dialect).
    Insert("users").
    Columns("name", "email").
    Values("John Doe", "john@example.com").
    Values("Jane Doe", "jane@example.com").
    Build()
```

### From Map

```go
data := map[string]any{
    "name":  "John Doe",
    "email": "john@example.com",
    "age":   30,
}

sql, args := query.New(dialect).
    Insert("users").
    ValuesMap(data).
    Build()
```

### RETURNING Clause

```go
// INSERT INTO users (...) VALUES (...) RETURNING id, created_at
sql, args := query.New(dialect).
    Insert("users").
    Columns("name", "email").
    Values("John Doe", "john@example.com").
    Returning("id", "created_at"). // PostgreSQL & SQLite only
    Build()
```

## UPDATE Queries

### Basic UPDATE

```go
// UPDATE users SET name = 'Jane', email = 'jane@...' WHERE id = 123
sql, args := query.New(dialect).
    Update("users").
    Set("name", "Jane Doe").
    Set("email", "jane@example.com").
    Where(query.Eq("id", 123)).
    Build()
```

### Update from Map

```go
updates := map[string]any{
    "name":       "Jane Doe",
    "email":      "jane@example.com",
    "updated_at": time.Now(),
}

sql, args := query.New(dialect).
    Update("users").
    SetMap(updates).
    Where(query.Eq("id", 123)).
    Build()
```

### Conditional UPDATE

```go
// UPDATE posts SET status = 'archived'
// WHERE last_updated < '2024-01-01' AND view_count < 100
sql, args := query.New(dialect).
    Update("posts").
    Set("status", "archived").
    Where(query.And(
        query.Lt("last_updated", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
        query.Lt("view_count", 100),
    )).
    Build()
```

## DELETE Queries

### Basic DELETE

```go
// DELETE FROM users WHERE id = 123
sql, args := query.New(dialect).
    Delete("users").
    Where(query.Eq("id", 123)).
    Build()
```

### Conditional DELETE

```go
// DELETE FROM sessions WHERE expires_at < NOW() AND user_id IS NULL
sql, args := query.New(dialect).
    Delete("sessions").
    Where(query.Lt("expires_at", time.Now())).
    Where(query.IsNull("user_id")).
    Build()
```

## Conditions

### Comparison Operators

```go
query.Eq("status", "active")           // status = 'active'
query.Ne("role", "guest")              // role != 'guest'
query.Gt("age", 18)                    // age > 18
query.Gte("score", 100)                // score >= 100
query.Lt("price", 50)                  // price < 50
query.Lte("quantity", 10)              // quantity <= 10
```

### Pattern Matching

```go
query.Like("name", "%John%")           // name LIKE '%John%'
query.NotLike("email", "%spam%")       // email NOT LIKE '%spam%'
query.ILike("name", "%john%")          // name ILIKE '%john%' (case-insensitive)
```

### NULL Checks

```go
query.IsNull("deleted_at")             // deleted_at IS NULL
query.IsNotNull("email")               // email IS NOT NULL
```

### Range and Sets

```go
query.In("status", "active", "pending")              // status IN ('active', 'pending')
query.NotIn("role", "banned", "suspended")           // role NOT IN ('banned', 'suspended')
query.Between("age", 18, 65)                         // age BETWEEN 18 AND 65
```

### Logical Operators

```go
// AND
query.And(
    query.Eq("status", "active"),
    query.Gt("age", 18),
)

// OR
query.Or(
    query.Eq("role", "admin"),
    query.Eq("role", "moderator"),
)

// NOT
query.Not(query.Eq("status", "banned"))

// Complex combinations
query.And(
    query.Eq("status", "active"),
    query.Or(
        query.Eq("role", "admin"),
        query.Eq("role", "moderator"),
    ),
    query.IsNotNull("email"),
)
```

### Column Comparison

```go
query.ColEq("created_at", "updated_at")  // created_at = updated_at
query.ColGt("price", "cost")             // price > cost
```

### Raw SQL (Escape Hatch)

```go
// Use when you need database-specific features
query.Raw("age > ? AND status = ?", 18, "active")
```

## Advanced Features

### CASE Expressions

```go
// CASE
//   WHEN age > 65 THEN 'senior'
//   WHEN age > 18 THEN 'adult'
//   ELSE 'minor'
// END as age_group
sql, args := query.New(dialect).
    Select("name").
    SelectExpr(
        query.Case(nil).
            WhenCond(query.Gt("age", 65), query.Lit("senior")).
            WhenCond(query.Gt("age", 18), query.Lit("adult")).
            Else(query.Lit("minor")).
            End().
            As("age_group"),
    ).
    From("users").
    Build()
```

### Window Functions

```go
// SELECT
//   name,
//   salary,
//   RANK() OVER (PARTITION BY department ORDER BY salary DESC) as rank
// FROM employees
sql, args := query.New(dialect).
    Select("name", "salary").
    SelectExpr(
        query.Rank(
            query.Window().
                PartitionBy(query.Col("department")).
                OrderBy(query.Col("salary"), query.DESC),
        ).As("rank"),
    ).
    From("employees").
    Build()
```

### Common Table Expressions (CTEs)

```go
// WITH active_users AS (
//   SELECT * FROM users WHERE status = 'active'
// )
// SELECT * FROM active_users WHERE age > 18
cte := query.New(dialect).
    Select("*").
    From("users").
    Where(query.Eq("status", "active"))

sql, args := query.New(dialect).
    WithCTE("active_users", cte).
    Select("*").
    From("active_users").
    Where(query.Gt("age", 18)).
    Build()
```

### Recursive CTEs

```go
// WITH RECURSIVE tree AS (
//   SELECT id, parent_id, name FROM categories WHERE parent_id IS NULL
//   UNION ALL
//   SELECT c.id, c.parent_id, c.name
//   FROM categories c INNER JOIN tree t ON c.parent_id = t.id
// )
// SELECT * FROM tree
baseCase := query.New(dialect).
    Select("id", "parent_id", "name").
    From("categories").
    Where(query.IsNull("parent_id"))

recursiveCase := query.New(dialect).
    Select("c.id", "c.parent_id", "c.name").
    From("categories").
    As("c").
    Join("tree", query.ColEq("c.parent_id", "t.id")).
    JoinAs("tree", "t", query.ColEq("c.parent_id", "t.id"))

sql, args := query.New(dialect).
    WithRecursiveCTE("tree", baseCase).
    // Add union handling for recursive part
    Select("*").
    From("tree").
    Build()
```

## Integration with Hooks

The query builder integrates seamlessly with GoREST's hook system, allowing you to modify queries without string manipulation.

### Modifying SELECT Queries

```go
type PostHooks struct {
    hooks.NoOpHooks[Post]
}

func (h *PostHooks) ModifySelectQuery(
    ctx context.Context,
    operation hooks.Operation,
    builder *query.SelectBuilder,
) (*query.SelectBuilder, bool) {
    // Add filter for unauthenticated users
    if !isAuthenticated(ctx) {
        builder = builder.Where(query.Eq("status", "published"))
        return builder, true
    }
    return builder, false
}
```

### Modifying UPDATE Queries

```go
func (h *PostHooks) ModifyUpdateQuery(
    ctx context.Context,
    operation hooks.Operation,
    id any,
    model *Post,
    builder *query.UpdateBuilder,
) (*query.UpdateBuilder, bool) {
    // Add tenant filter for multi-tenancy
    tenantID := getTenantID(ctx)
    builder = builder.Where(query.Eq("tenant_id", tenantID))
    return builder, true
}
```

### Modifying DELETE Queries

```go
func (h *PostHooks) ModifyDeleteQuery(
    ctx context.Context,
    operation hooks.Operation,
    id any,
    builder *query.DeleteBuilder,
) (*query.DeleteBuilder, bool) {
    // Soft delete instead of hard delete
    builder = builder.Where(query.IsNull("deleted_at"))
    return builder, true
}
```

## Database Compatibility

The query builder automatically generates correct SQL for your database:

| Feature | PostgreSQL | MySQL | SQLite |
|---------|-----------|-------|--------|
| Placeholders | `$1, $2` | `?, ?` | `?, ?` |
| RETURNING | ✅ | ❌ | ✅ |
| ILIKE | ✅ | ❌ (uses LOWER) | ❌ (uses LOWER) |
| Window Functions | ✅ | ✅ | ✅ |
| CTEs | ✅ | ✅ | ✅ |
| Recursive CTEs | ✅ | ✅ | ✅ |
| Full Outer Join | ✅ | ❌ | ❌ |

## Security

The query builder includes multiple layers of security protection to prevent SQL injection and other attacks.

### 1. Automatic Parameterization

All user-provided values are automatically parameterized, not concatenated into SQL strings:

```go
// SAFE - Value is parameterized
query.Where(query.Eq("email", userInput))
// Generates: WHERE "email" = $1
// Args: [userInput]

// NEVER manually concatenate:
// query.Raw(fmt.Sprintf("email = '%s'", userInput)) // UNSAFE!
```

### 2. Identifier Escaping

Table and column names are automatically escaped for each database:

```go
// All identifiers are escaped
sql, _ := query.New(dialect).
    Select("user.id", "user.name").   // Escaped: "user"."id", "user"."name"
    From("users").                     // Escaped: "users"
    Build()
```

**Protection against identifier injection:**
- PostgreSQL: Doubles internal quotes (`"` → `""`)
- MySQL: Doubles internal backticks (`` ` `` → ``` `` ```)
- SQLite: Doubles internal quotes (`"` → `""`)

```go
// Malicious input is neutralized
malicious := `email" FROM users WHERE admin=true; --`
query.Select(malicious).From("users").Build()
// Generates: SELECT "email"" FROM users WHERE admin=true; --" FROM "users"
// Database treats entire string as column name → query fails safely
```

### 3. Input Validation

The query builder validates inputs at `Build()` time and returns errors for invalid or dangerous inputs.

#### Identifier Validation

Identifiers must follow standard SQL rules:
- Start with letter or underscore
- Contain only letters, digits, underscores
- Maximum 63 characters (PostgreSQL limit)
- Cannot be SQL reserved words

```go
// Valid identifiers
builder.From("users")           // ✅
builder.From("user_profiles")   // ✅
builder.From("_temp")           // ✅

// Invalid identifiers return errors
builder.From("123users")        // ❌ Cannot start with digit
builder.From("user-profiles")   // ❌ Hyphen not allowed
builder.From("SELECT")          // ❌ Reserved word
builder.From(strings.Repeat("a", 64)) // ❌ Too long

sql, args, err := builder.Build()
if err != nil {
    // Handle validation error
    log.Printf("Invalid query: %v", err)
}
```

**Reserved words blocked:**
```go
// These identifiers are blocked to prevent confusion
SELECT, INSERT, UPDATE, DELETE, FROM, WHERE, JOIN, ORDER, GROUP,
UNION, CREATE, DROP, TABLE, INDEX, VIEW, AND, OR, NOT, IN, EXISTS,
BETWEEN, LIKE, IS, NULL, TRUE, FALSE, CASE, WHEN, THEN, ELSE, END,
WITH, RECURSIVE, OVER, PARTITION, ROWS, RANGE, ...
```

#### LIMIT/OFFSET Validation

Prevents resource exhaustion attacks:

```go
// Reasonable limits
builder.Limit(100)              // ✅
builder.Offset(1000)            // ✅

// Excessive values return errors
builder.Limit(999999)           // ❌ Exceeds max (10,000)
builder.Offset(10000000)        // ❌ Exceeds max (1,000,000)

sql, args, err := builder.Build()
// err: "LIMIT 999999 exceeds maximum allowed value of 10000"
```

**Configurable limits:**
```go
query.MaxLimitValue = 10000      // Max LIMIT value
query.MaxOffsetValue = 1000000   // Max OFFSET value
```

#### Window Frame Validation

Validates window frame syntax to prevent injection:

```go
// Valid frames
query.Window().Frame("ROWS UNBOUNDED PRECEDING")          // ✅
query.Window().Frame("RANGE BETWEEN 1 PRECEDING AND 1 FOLLOWING") // ✅

// Invalid frames are blocked
query.Window().Frame("ROWS; DROP TABLE users--")          // ❌
query.Window().Frame("INVALID SYNTAX")                     // ❌
```

### 4. Subquery Depth Limiting

Prevents deeply nested subqueries that could cause performance issues or stack overflow:

```go
query.MaxSubqueryDepth = 3  // Maximum nesting depth

// This is safe (depth 1)
subquery := query.New(dialect).Select("id").From("banned_users")
query.New(dialect).Select("*").From("users").Where(query.InSubquery("id", subquery"))

// Deep nesting would be rejected (not yet implemented)
```

### 5. Error Handling

Always check errors from `Build()`:

```go
sql, args, err := builder.Build()
if err != nil {
    // Validation failed - do NOT execute query
    return fmt.Errorf("invalid query: %w", err)
}

// Safe to execute
rows, err := db.Query(ctx, sql, args...)
```

### 6. Security Best Practices

#### ✅ DO: Use the query builder for all queries
```go
// Type-safe, parameterized, validated
sql, args, _ := query.New(dialect).
    Select("*").
    From("users").
    Where(query.Eq("email", userEmail)).
    Build()
```

#### ❌ DON'T: Concatenate user input
```go
// NEVER DO THIS - vulnerable to SQL injection
sql := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", userEmail)
```

#### ✅ DO: Validate business logic separately
```go
// The query builder validates SQL structure, not business rules
if !isValidEmail(userEmail) {
    return errors.New("invalid email format")
}
sql, args, _ := query.New(dialect).Select("*").From("users").Where(query.Eq("email", userEmail")).Build()
```

#### ❌ DON'T: Trust client-provided table/column names without validation
```go
// If accepting dynamic table names, maintain a whitelist
allowedTables := map[string]bool{"users": true, "posts": true}
if !allowedTables[tableName] {
    return errors.New("invalid table name")
}
sql, args, _ := query.New(dialect).Select("*").From(tableName).Build()
```

#### ✅ DO: Use Raw() sparingly and carefully
```go
// Raw() bypasses validation - use only when necessary
// Still parameterize values:
query.Raw("age > ? AND status = ?", 18, "active")  // ✅ Values parameterized

// Never concatenate into Raw():
// query.Raw(fmt.Sprintf("age > %d", userInput))    // ❌ DANGEROUS
```

### Special Warning: Raw SQL Functions

The query builder provides `Raw()` and `RawExpr()` functions for cases where the builder doesn't support specific SQL syntax. **These functions bypass all security protections and must be used with extreme caution.**

#### ⚠️ High Risk Functions

- `query.Raw(sql, args...)` - Raw WHERE condition
- `query.RawExpr(sql)` - Raw SELECT expression

#### Rules for Safe Usage

1. **NEVER use with user input**
   ```go
   // ❌ DANGEROUS - SQL INJECTION!
   query.Where(query.Raw(fmt.Sprintf("name = '%s'", userName)))

   // ✅ SAFE - Use type-safe builders
   query.Where(query.Eq("name", userName))
   ```

2. **ALWAYS use placeholders for values**
   ```go
   // ❌ DANGEROUS - SQL INJECTION!
   query.Raw(fmt.Sprintf("age > %d", minAge))

   // ✅ SAFE - Use placeholders
   query.Raw("age > ?", minAge)
   ```

3. **ONLY use for trusted, static SQL**
   ```go
   // ✅ SAFE - Static SQL function
   query.Raw("created_at > NOW() - INTERVAL '1 day'")

   // ✅ SAFE - Database function with parameters
   query.Raw("ST_Distance(point, ?) < ?", userLocation, maxDistance)
   ```

#### When to Use Raw()

Raw SQL should be a last resort. Use it only for:

- Complex database-specific functions not supported by the builder
- Performance-critical hand-optimized SQL
- Temporary workarounds (create an issue to add proper builder support)

#### Code Review Checklist

During security audits, review ALL uses of `Raw()` and `RawExpr()`:

- [ ] Is user input being passed to Raw()?
- [ ] Are values properly parameterized with `?` placeholders?
- [ ] Could this be replaced with type-safe builder methods?
- [ ] Is there a comment explaining why Raw() is necessary?

**Remember:** If you find yourself using Raw() frequently, consider contributing builder support for that SQL feature instead.

### 7. Security Testing

Test your queries with malicious inputs:

```go
func TestQuerySecurityValidator(t *testing.T) {
    dialect := &postgres.PostgresDialect{}

    // Test reserved word blocking
    builder := query.New(dialect).Select("*").From("SELECT")
    _, _, err := builder.Build()
    if err == nil {
        t.Error("Expected error for reserved word, got nil")
    }

    // Test identifier injection
    malicious := `users" WHERE admin=true; --`
    builder = query.New(dialect).Select("*").From(malicious)
    _, _, err = builder.Build()
    if err == nil {
        t.Error("Expected error for malicious identifier")
    }

    // Test LIMIT overflow
    builder = query.New(dialect).Select("*").From("users").Limit(999999)
    _, _, err = builder.Build()
    if err == nil {
        t.Error("Expected error for excessive LIMIT")
    }
}
```

### 8. Defense in Depth

The query builder provides multiple overlapping security layers:

1. **Parameterization** - Separates data from SQL structure
2. **Identifier escaping** - Neutralizes injection attempts in table/column names
3. **Input validation** - Rejects invalid or suspicious inputs before query generation
4. **Length limits** - Prevents excessively long identifiers
5. **Reserved word blocking** - Prevents confusion with SQL keywords
6. **Value bounds** - Limits LIMIT/OFFSET to reasonable ranges

Even if one layer fails, others provide protection.

### 9. Reporting Security Issues

If you discover a security vulnerability:

1. **DO NOT** create a public GitHub issue
2. Email security concerns to: [security contact]
3. Include proof-of-concept code
4. Allow time for a fix before public disclosure

## Best Practices

### 1. Reuse Query Components

```go
// Define common filters as functions
func activeUsersFilter() query.Condition {
    return query.And(
        query.Eq("status", "active"),
        query.IsNotNull("email"),
    )
}

// Reuse across queries
sql1, _ := query.New(dialect).Select("*").From("users").Where(activeUsersFilter()).Build()
sql2, _ := query.New(dialect).Delete("users").Where(query.Not(activeUsersFilter())).Build()
```

### 2. Use Expressions for Complex Selects

```go
// Instead of raw column names
builder.SelectExpr(
    query.Col("id"),
    query.Col("name"),
    query.Concat(query.Col("first_name"), query.Lit(" "), query.Col("last_name")).As("full_name"),
)
```

### 3. Leverage Type Safety

```go
// Compile-time error prevention
const (
    StatusActive   = "active"
    StatusInactive = "inactive"
)

builder.Where(query.Eq("status", StatusActive)) // Type-safe constant
```

### 4. Test Queries

```go
func TestUserQuery(t *testing.T) {
    dialect := &postgres.PostgresDialect{}
    sql, args := query.New(dialect).
        Select("*").
        From("users").
        Where(query.Eq("status", "active")).
        Build()

    expected := `SELECT * FROM "users" WHERE "status" = $1`
    if sql != expected {
        t.Errorf("Expected %s, got %s", expected, sql)
    }

    if len(args) != 1 || args[0] != "active" {
        t.Errorf("Expected args [active], got %v", args)
    }
}
```

## Complete Example

Here's a real-world example combining multiple features:

```go
package main

import (
    "context"
    "time"

    "github.com/nicolasbonnici/gorest/database"
    "github.com/nicolasbonnici/gorest/query"
)

type UserReport struct {
    Name        string
    Email       string
    PostCount   int
    LastPost    time.Time
}

func GetActiveUserReport(db database.Database, minPosts int) ([]UserReport, error) {
    ctx := context.Background()
    builder := query.New(db.Dialect())

    sql, args := builder.
        Select(
            "u.name",
            "u.email",
        ).
        SelectExpr(
            query.Count(query.Col("p.id")).As("post_count"),
            query.Max(query.Col("p.created_at")).As("last_post"),
        ).
        From("users").
        As("u").
        Join("posts", query.ColEq("u.id", "p.user_id")).
        JoinAs("posts", "p", query.ColEq("u.id", "p.user_id")).
        Where(query.And(
            query.Eq("u.status", "active"),
            query.Eq("p.status", "published"),
            query.Gt("p.created_at", time.Now().AddDate(0, -6, 0)),
        )).
        GroupBy("u.id", "u.name", "u.email").
        HavingExpr(query.Gt(query.Count(query.Col("p.id")), minPosts)).
        OrderByExpr(query.Count(query.Col("p.id")), query.DESC).
        Limit(100).
        Build()

    rows, err := db.Query(ctx, sql, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var reports []UserReport
    for rows.Next() {
        var r UserReport
        err := rows.Scan(&r.Name, &r.Email, &r.PostCount, &r.LastPost)
        if err != nil {
            return nil, err
        }
        reports = append(reports, r)
    }

    return reports, rows.Err()
}
```

## Further Reading

- **[Full API Reference →](QUERY_BUILDER_API.md)** - Complete API documentation
- **[Hooks System →](HOOKS.md)** - Integrate query builder with hooks
- **[Database Package →](database/)** - Database abstraction layer

## Migration from Raw SQL

**Before (raw SQL):**
```go
query := fmt.Sprintf("SELECT * FROM users WHERE status = %s AND age > %s",
    dialect.Placeholder(1), dialect.Placeholder(2))
args := []any{"active", 18}
```

**After (query builder):**
```go
query, args := query.New(dialect).
    Select("*").
    From("users").
    Where(query.Eq("status", "active")).
    Where(query.Gt("age", 18)).
    Build()
```

**Benefits:**
- ✅ No string concatenation
- ✅ Type-safe
- ✅ Database-agnostic
- ✅ Compile-time checks
- ✅ Better IDE support
- ✅ Easier to test
- ✅ SQL injection prevention

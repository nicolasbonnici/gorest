# GoREST Fixtures Package

A comprehensive fixture management system for GoREST that eliminates duplicate test setup code and provides centralized test data management.

## Features

- **Programmatic Fixtures**: Load fixtures from Go structs with type safety
- **File-based Fixtures**: Load fixtures from YAML/JSON files for large datasets
- **Fluent API**: Chainable methods for cleaner test code
- **Auto-cleanup**: Automatic cleanup after tests with defer pattern
- **Transaction Support**: Test isolation using database transactions
- **Multi-database Support**: Works with PostgreSQL, MySQL, and SQLite
- **Dependency Ordering**: Handle foreign key constraints with ordered cleanup

## Installation

The fixtures package is part of GoREST and requires no additional installation.

## Quick Start

### Define Your Models

First, define models that implement the `crud.Model` interface:

```go
type User struct {
    ID        string `db:"id"`
    Firstname string `db:"firstname"`
    Lastname  string `db:"lastname"`
    Email     string `db:"email"`
}

func (u User) TableName() string { return "users" }

type Todo struct {
    ID      string `db:"id"`
    UserID  string `db:"user_id"`
    Title   string `db:"title"`
    Content string `db:"content"`
}

func (t Todo) TableName() string { return "todo" }
```

### Basic Usage

```go
func TestUsers(t *testing.T) {
    db := setupTestDB(t)

    users := []User{
        {ID: "user-1", Firstname: "Alice", Lastname: "Smith", Email: "alice@test.com"},
        {ID: "user-2", Firstname: "Bob", Lastname: "Jones", Email: "bob@test.com"},
    }

    loader := fixtures.New(db)
    _, err := fixtures.Load(loader, "users", users)
    if err != nil {
        t.Fatalf("failed to load fixtures: %v", err)
    }

    // Your test code here...
}
```

### Using the Fluent Builder API

```go
func TestWithBuilder(t *testing.T) {
    db := setupTestDB(t)

    users := []User{
        {ID: "user-1", Firstname: "Alice", Lastname: "Smith", Email: "alice@test.com"},
    }

    todos := []Todo{
        {ID: "todo-1", UserID: "user-1", Title: "Task 1", Content: "Content"},
    }

    builder := fixtures.NewBuilderWithT(t, db)
    fixtures.LoadBuilder(builder, "users", users)
    fixtures.LoadBuilder(builder, "todos", todos)
    builder.Cleanup()

    // Fixtures auto-cleanup after test via t.Cleanup()
}
```

## Advanced Usage

### Transaction Support for Test Isolation

```go
func TestWithTransaction(t *testing.T) {
    db := setupTestDB(t)

    users := []User{
        {ID: "user-1", Firstname: "Alice", Lastname: "Smith", Email: "alice@test.com"},
    }

    builder := fixtures.NewBuilder(db).WithTransaction()
    fixtures.LoadBuilder(builder, "users", users)

    // Your test code here...

    // Rollback to clean up
    builder.Rollback()
}
```

### Loading from YAML Files

```go
func TestLoadFromYAML(t *testing.T) {
    db := setupTestDB(t)

    var users []User

    fixtures.NewBuilder(db).
        LoadFromYAML("users", "testdata/users.yaml", &users).
        Cleanup()

    // Test with loaded users...
}
```

Example YAML file:
```yaml
- id: user-1
  firstname: Alice
  lastname: Smith
  email: alice@test.com
```

### Loading from JSON Files

```go
func TestLoadFromJSON(t *testing.T) {
    db := setupTestDB(t)

    var users []User

    fixtures.NewBuilder(db).
        LoadFromJSON("users", "testdata/users.json", &users).
        Cleanup()

    // Test with loaded users...
}
```

Example JSON file:
```json
[
  {
    "id": "user-1",
    "firstname": "Alice",
    "lastname": "Smith",
    "email": "alice@test.com"
  }
]
```

### Handling Foreign Key Constraints

Use `CleanupOrdered` to delete fixtures in the correct order:

```go
func TestWithForeignKeys(t *testing.T) {
    db := setupTestDB(t)

    loader := fixtures.New(db).EnableCleanup()

    users := []User{
        {ID: "user-1", Firstname: "Alice", Lastname: "Smith", Email: "alice@test.com"},
    }
    fixtures.Load(loader, "users", users)

    todos := []Todo{
        {ID: "todo-1", UserID: "user-1", Title: "Task", Content: "Content"},
    }
    fixtures.Load(loader, "todos", todos)

    defer func() {
        // Delete in reverse dependency order (child before parent)
        fixtures.CleanupOrdered(loader, []string{"users", "todos"})
    }()

    // Test code...
}
```

### Cleanup Strategies

The package supports multiple cleanup strategies:

1. **Delete** (default): Deletes all loaded fixtures by ID
2. **Truncate**: Truncates all tables that had fixtures loaded
3. **Rollback**: Rolls back the transaction (requires transaction support)

```go
// Using truncate for faster cleanup
fixtures.CleanupWithStrategy(loader, fixtures.CleanupTruncate)

// Using rollback
fixtures.CleanupWithStrategy(loader, fixtures.CleanupRollback)
```

## API Reference

### Core Functions

#### `New(db database.Database) *Loader`
Creates a new fixture loader with the given database.

#### `Load[T crud.Model](loader *Loader, name string, fixtures []T) (*Loader, error)`
Loads fixtures from Go structs. IDs are preserved if provided, otherwise auto-generated by the database.

#### `LoadBuilder[T crud.Model](builder *Builder, name string, fixtures []T) *Builder`
Loads fixtures using the builder pattern for fluent API.

#### `Cleanup(loader *Loader) error`
Cleans up all loaded fixtures if cleanup is enabled.

#### `CleanupOrdered(loader *Loader, order []string) error`
Cleans up fixtures in a specific order to respect foreign key constraints.

### Loader Methods

- `WithContext(ctx context.Context) *Loader` - Sets the context for database operations
- `WithTransaction() (*Loader, error)` - Enables transaction support
- `Commit() error` - Commits the current transaction
- `Rollback() error` - Rolls back the current transaction
- `EnableCleanup() *Loader` - Marks the loader to cleanup fixtures after use
- `LoadFromYAML(name, filePath string, target interface{}) (*Loader, error)` - Loads from YAML
- `LoadFromJSON(name, filePath string, target interface{}) (*Loader, error)` - Loads from JSON
- `Get(name string) ([]interface{}, bool)` - Retrieves loaded fixtures by name
- `GetLoadedFixtures() []string` - Returns all loaded fixture names

### Builder API

#### `NewBuilder(db database.Database) *Builder`
Creates a new builder with fluent API.

#### `NewBuilderWithT(t *testing.T, db database.Database) *Builder`
Creates a builder integrated with testing.T for automatic error reporting and cleanup.

#### Builder Methods
- `WithContext(ctx context.Context) *Builder`
- `WithTransaction() *Builder`
- `LoadFromYAML(name, filePath string, target interface{}) *Builder`
- `LoadFromJSON(name, filePath string, target interface{}) *Builder`
- `Commit() *Builder`
- `Rollback() *Builder`
- `Cleanup() *Builder`
- `Get(name string) ([]interface{}, bool)`
- `Error() error` - Returns any error that occurred during operations
- `Loader() *Loader` - Returns the underlying loader instance

## Best Practices

1. **Use the Builder API** for cleaner, more readable test code
2. **Enable auto-cleanup** to ensure test isolation
3. **Use transactions** for tests that modify data to enable fast rollbacks
4. **Specify IDs explicitly** when you need to reference fixtures across different loads
5. **Use ordered cleanup** when dealing with foreign key constraints
6. **Use file-based fixtures** for large datasets or shared test data

## Testing

Run the fixture tests:
```bash
go test -v ./fixtures/...
```

Check coverage:
```bash
go test -coverprofile=coverage.out ./fixtures/...
go tool cover -html=coverage.out
```

Current test coverage: **87.5%**

## Example: Complete Test Suite

```go
func TestUserTodoWorkflow(t *testing.T) {
    db := setupTestDB(t)

    builder := fixtures.NewBuilderWithT(t, db)

    users := []User{
        {ID: "alice", Firstname: "Alice", Lastname: "Smith", Email: "alice@test.com"},
        {ID: "bob", Firstname: "Bob", Lastname: "Jones", Email: "bob@test.com"},
    }

    todos := []Todo{
        {ID: "todo-1", UserID: "alice", Title: "Alice's Task", Content: "Content 1"},
        {ID: "todo-2", UserID: "bob", Title: "Bob's Task", Content: "Content 2"},
    }

    fixtures.LoadBuilder(builder, "users", users)
    fixtures.LoadBuilder(builder, "todos", todos)
    builder.Cleanup()

    // Run your tests...
    t.Run("can create todos", func(t *testing.T) {
        // Test implementation
    })

    t.Run("can list todos by user", func(t *testing.T) {
        // Test implementation
    })

    // Cleanup happens automatically
}
```

## License

Part of the GoREST project. See main repository for license information.

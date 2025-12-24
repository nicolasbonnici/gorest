# GoREST Database Migrations

A production-ready database migration system with support for multiple databases, plugin migrations, and comprehensive safety features.

## Features

- ✅ **Go-Based Migrations**: Define migrations in Go code using the Database Abstraction Layer (DAL)
- ✅ **Multi-Database Support**: PostgreSQL, MySQL, SQLite - write once, run anywhere
- ✅ **Timestamp Naming**: Millisecond-precision timestamps prevent conflicts between developers
- ✅ **Plugin System Integration**: Each plugin can maintain its own migrations
- ✅ **Checksum Verification**: Prevents migration drift between environments
- ✅ **Advisory Locking**: Prevents concurrent execution
- ✅ **Dirty Database Detection**: Tracks and blocks on failed migrations
- ✅ **Transaction Safety**: Automatic rollback on failure
- ✅ **Dependency Resolution**: Plugin migrations run in correct order
- ✅ **SQL File Support**: Legacy support for .sql files (deprecated)

## Quick Start

### 1. Generate a New Migration

Use the migration generator to create a timestamped migration file:

```bash
go run github.com/nicolasbonnici/gorest/migrations/cmd/generate-migration -name=create_users_table

# Output:
# ✓ Created migration: 20251222204530123_create_users_table.go
```

This creates a file like `20251222204530123_create_users_table.go` with the timestamp format:
- `YYYYMMDDHHMMSSmmm` (17 digits with milliseconds)
- Example: `20251222204530123` = December 22, 2025 at 20:45:30.123

### 2. Define Your Migration

Edit the generated file to implement your migration logic:

```go
package migrations

import (
    "context"

    "github.com/nicolasbonnici/gorest/database"
    "github.com/nicolasbonnici/gorest/migrations"
)

func Migration_20251222204530123_CreateUsersTable() migrations.Migration {
    return migrations.NewGoMigration("20251222204530123", "create_users_table").
        Up(func(ctx context.Context, db database.Database) error {
            return migrations.SQL(ctx, db, migrations.DialectSQL{
                Postgres: `CREATE TABLE users (
                    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                    email TEXT UNIQUE NOT NULL,
                    password TEXT NOT NULL,
                    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
                )`,
                MySQL: `CREATE TABLE users (
                    id CHAR(36) PRIMARY KEY,
                    email VARCHAR(255) UNIQUE NOT NULL,
                    password VARCHAR(255) NOT NULL,
                    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
                ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
                SQLite: `CREATE TABLE users (
                    id TEXT PRIMARY KEY,
                    email TEXT UNIQUE NOT NULL,
                    password TEXT NOT NULL,
                    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
                )`,
            })
        }).
        Down(func(ctx context.Context, db database.Database) error {
            return migrations.DropTableIfExists(ctx, db, "users")
        }).
        Build()
}
```

### 3. Register Migrations

Create a migration source with your migrations:

```go
package main

import (
    "context"
    "log"

    "github.com/nicolasbonnici/gorest/database"
    "github.com/nicolasbonnici/gorest/migrations"
)

func main() {
    db, err := database.Open("postgres", "postgres://localhost/mydb")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Build migration source using the fluent API
    builder := migrations.NewMigrationBuilder("app")

    // Add your migrations
    builder.Add(
        "20251222204530123",
        "create_users_table",
        createUsersUp,
        createUsersDown,
    )

    builder.Add(
        "20251222204530124",
        "add_users_email_index",
        addEmailIndexUp,
        addEmailIndexDown,
    )

    source := builder.Build()

    // Create migrator
    migrator := migrations.NewMigrator(db, source)

    // Run all pending migrations
    if err := migrator.Up(context.Background()); err != nil {
        log.Fatal(err)
    }

    log.Println("Migrations applied successfully")
}

func createUsersUp(ctx context.Context, db database.Database) error {
    return migrations.SQL(ctx, db, migrations.DialectSQL{
        Postgres: `CREATE TABLE users (...)`,
        MySQL:    `CREATE TABLE users (...)`,
        SQLite:   `CREATE TABLE users (...)`,
    })
}

func createUsersDown(ctx context.Context, db database.Database) error {
    return migrations.DropTableIfExists(ctx, db, "users")
}

func addEmailIndexUp(ctx context.Context, db database.Database) error {
    return migrations.CreateIndex(ctx, db, "idx_users_email", "users", "email")
}

func addEmailIndexDown(ctx context.Context, db database.Database) error {
    return migrations.DropIndex(ctx, db, "idx_users_email", "users")
}
```

### 4. Run Migrations

```go
ctx := context.Background()

// Apply all pending migrations
migrator.Up(ctx)

// Apply next pending migration
migrator.UpOne(ctx)

// Apply migrations up to specific version
migrator.UpTo(ctx, "20251222204530123")

// Revert last migration
migrator.Down(ctx)

// Revert to specific version
migrator.DownTo(ctx, "20251222204530123")

// Check migration status
statuses, _ := migrator.Status(ctx)
for _, s := range statuses {
    fmt.Printf("[%s] %s: %s\n", s.Migration.Source, s.Migration.FullName(), s.Status)
}
```

## Migration Helpers

The migration system provides helper functions for common operations:

### Table Operations

```go
// Create table (dialect-aware)
migrations.CreateTableIfNotExists(ctx, db, "users", "id TEXT PRIMARY KEY, email TEXT")

// Drop table
migrations.DropTableIfExists(ctx, db, "users")
```

### Column Operations

```go
// Add column
migrations.AddColumn(ctx, db, "users", "name TEXT")

// Drop column
migrations.DropColumn(ctx, db, "users", "name")
```

### Index Operations

```go
// Create index
migrations.CreateIndex(ctx, db, "idx_users_email", "users", "email")

// Drop index
migrations.DropIndex(ctx, db, "idx_users_email", "users")
```

### Dialect-Specific SQL

```go
migrations.SQL(ctx, db, migrations.DialectSQL{
    Postgres: `postgres SQL here`,
    MySQL:    `mysql SQL here`,
    SQLite:   `sqlite SQL here`,
})

// All fields are optional - specify only what you need
migrations.SQL(ctx, db, migrations.DialectSQL{
    Postgres: `postgres-only SQL here`,
})
```

## Migration Builder

Use the fluent builder API for clean migration code:

```go
builder := migrations.NewMigrationBuilder("app")

builder.Add(
    "20251222204530123",
    "create_users",
    func(ctx context.Context, db database.Database) error {
        return migrations.CreateTableIfNotExists(ctx, db, "users", "id TEXT PRIMARY KEY")
    },
    func(ctx context.Context, db database.Database) error {
        return migrations.DropTableIfExists(ctx, db, "users")
    },
)

// For SQL-only migrations (simple cases)
builder.AddSQL(
    "20251222204530124",
    "add_index",
    "CREATE INDEX idx_users_email ON users(email)",
    "DROP INDEX idx_users_email",
)

source := builder.Build()
```

## Examples

See `migrations/examples.go` for comprehensive examples including:

1. Creating tables with dialect-aware SQL
2. Adding indexes
3. Adding/dropping columns
4. Foreign key relationships
5. Data migrations
6. Complex multi-step migrations

## Plugin Migrations

Plugins can provide their own migrations:

```go
package auth

import (
    "github.com/nicolasbonnici/gorest/migrations"
    "github.com/nicolasbonnici/gorest/plugin"
)

type AuthPlugin struct {
    db database.Database
}

// MigrationSource implements plugin.MigrationProvider
func (p *AuthPlugin) MigrationSource() interface{} {
    builder := migrations.NewMigrationBuilder("auth")

    builder.Add(
        "20251222204530200",
        "create_sessions_table",
        p.createSessionsUp,
        p.createSessionsDown,
    )

    return builder.Build()
}

// MigrationDependencies returns dependencies
func (p *AuthPlugin) MigrationDependencies() []string {
    return []string{"app"} // Auth depends on app migrations
}

func (p *AuthPlugin) createSessionsUp(ctx context.Context, db database.Database) error {
    return migrations.SQL(ctx, db, migrations.DialectSQL{
        Postgres: `CREATE TABLE sessions (...)`,
        MySQL:    `CREATE TABLE sessions (...)`,
        SQLite:   `CREATE TABLE sessions (...)`,
    })
}

func (p *AuthPlugin) createSessionsDown(ctx context.Context, db database.Database) error {
    return migrations.DropTableIfExists(ctx, db, "sessions")
}
```

### Using Plugin Migrations

```go
db, _ := database.Open("postgres", "postgres://localhost/mydb")
defer db.Close()

// Core app migrations
appSource := buildAppMigrations()

// Collect plugin migrations
sources := []migrations.MigrationSource{appSource}

authPlugin := auth.NewPlugin()
if provider, ok := authPlugin.(plugin.MigrationProvider); ok {
    sources = append(sources, provider.MigrationSource().(migrations.MigrationSource))
}

// Create migrator with all sources
migrator := migrations.NewMigrator(db, sources...)

// Set dependencies
if provider, ok := authPlugin.(plugin.MigrationProvider); ok {
    deps := provider.MigrationDependencies()
    migrator.SetSourceDependencies("auth", deps)
}

// Run all migrations (app + plugins)
migrator.Up(context.Background())
```

## Timestamp Format

Migrations use millisecond-precision timestamps to prevent conflicts:

- **Format**: `YYYYMMDDHHMMSSmmm` (17 digits)
- **Example**: `20251222204530123`
  - Year: 2025
  - Month: 12 (December)
  - Day: 22
  - Hour: 20 (8 PM)
  - Minute: 45
  - Second: 30
  - Milliseconds: 123

**Benefits:**
- Chronological ordering
- No conflicts between developers
- Clear when migration was created
- Sortable as strings

**Generate timestamp:**
```bash
go run github.com/nicolasbonnici/gorest/migrations/cmd/generate-migration -name=my_migration
```

## API Reference

### Migrator Interface

```go
type Migrator interface {
    Up(ctx context.Context) error
    UpWithOptions(ctx context.Context, opts MigrationOptions) error
    UpOne(ctx context.Context) error
    UpTo(ctx context.Context, version string) error
    Down(ctx context.Context) error
    DownTo(ctx context.Context, version string) error
    Status(ctx context.Context) ([]MigrationStatus, error)
    Pending(ctx context.Context) ([]Migration, error)
    Validate(ctx context.Context) error
    DryRun(ctx context.Context) ([]Migration, error)
    Force(ctx context.Context, version, source string) error
    UpSource(ctx context.Context, sourceName string) error
    DownSource(ctx context.Context, sourceName string) error
}
```

### Migration Options

```go
type MigrationOptions struct {
    Transactional bool // Wrap all migrations in single transaction
    DryRun        bool // Show what would execute without executing
    StopOnError   bool // Stop on first error (default: true)
}

// Example: All-or-nothing migration
migrator.UpWithOptions(ctx, migrations.MigrationOptions{
    Transactional: true,
})
```

## Safety Features

### 1. Checksum Verification

Every migration is hashed. If modified after being applied, the system detects it:

```
CRITICAL: Migration app/20251222204530123_create_users has been modified!
Expected checksum: abc123...
Actual checksum:   def456...
```

### 2. Advisory Locking

Prevents concurrent migrations:
- **PostgreSQL**: Uses `pg_advisory_lock()`
- **MySQL**: Uses `GET_LOCK()` with 60s timeout
- **SQLite**: Uses EXCLUSIVE transactions

### 3. Dirty Database Detection

Blocks execution if migrations failed:

```
Database is in dirty state - 1 failed migration(s):
  - [app] 20251222204530123_create_users: syntax error

Fix the migration and retry, or use Force()
```

### 4. Transaction Rollback

Each migration runs in a transaction. On failure:
- SQL changes rolled back
- Migration marked as failed
- Execution stops

## Schema Migrations Table

Migrations are tracked in `schema_migrations`:

```sql
CREATE TABLE schema_migrations (
    version VARCHAR(17) NOT NULL,              -- Timestamp with milliseconds
    source VARCHAR(100) NOT NULL DEFAULT 'app',
    name VARCHAR(255) NOT NULL,
    checksum CHAR(64) NOT NULL,                -- SHA256 hash
    status VARCHAR(20) NOT NULL DEFAULT 'applied',
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    execution_time_ms INTEGER,
    executed_by VARCHAR(100),
    hostname VARCHAR(255),
    error_message TEXT,
    PRIMARY KEY (version, source)
);
```

**Query status:**
```sql
SELECT version, source, name, status, applied_at, execution_time_ms
FROM schema_migrations
ORDER BY applied_at DESC;
```

## Best Practices

### 1. Never Modify Applied Migrations

Once applied in any environment, **never modify** the migration. Create a new one instead.

❌ **Wrong:**
```go
// Already applied migration - DON'T MODIFY
builder.Add("20251222204530123", "create_users",
    func(ctx context.Context, db database.Database) error {
        // Adding a new column to already-applied migration
        return migrations.CreateTableIfNotExists(ctx, db, "users", "id TEXT, email TEXT, name TEXT")
    },
    // ...
)
```

✅ **Correct:**
```go
// New migration to add column
builder.Add("20251222204530456", "add_name_to_users",
    func(ctx context.Context, db database.Database) error {
        return migrations.AddColumn(ctx, db, "users", "name TEXT")
    },
    func(ctx context.Context, db database.Database) error {
        return migrations.DropColumn(ctx, db, "users", "name")
    },
)
```

### 2. Always Provide Down Migrations

Every migration must be reversible:

```go
builder.Add(version, "create_table",
    func(ctx context.Context, db database.Database) error {
        return migrations.CreateTableIfNotExists(ctx, db, "users", "...")
    },
    func(ctx context.Context, db database.Database) error {
        return migrations.DropTableIfExists(ctx, db, "users")
    },
)
```

### 3. Use Helpers for Portability

Prefer migration helpers over raw SQL:

✅ **Good:**
```go
migrations.CreateTableIfNotExists(ctx, db, "users", "id TEXT PRIMARY KEY")
migrations.CreateIndex(ctx, db, "idx_users_email", "users", "email")
```

❌ **Avoid:**
```go
db.Exec(ctx, "CREATE TABLE users (...)")  // May not work on all databases
```

### 4. Test Migrations

Test both up and down:
```go
// Test up
migrator.Up(ctx)

// Verify table exists
// ...

// Test down
migrator.Down(ctx)

// Verify table removed
// ...
```

### 5. Keep Migrations Focused

One logical change per migration:
- Create a table
- Add a column
- Create an index

### 6. Use Descriptive Names

```
✅ 20251222204530123_create_users_table
✅ 20251222204530456_add_email_index_to_users
❌ 20251222204530789_migration
❌ 20251222204531000_update
```

## Legacy: SQL File Migrations (Deprecated)

The system still supports SQL files for backward compatibility:

```
migrations/
├── 20250120143022_create_users.up.postgres.sql
├── 20250120143022_create_users.down.postgres.sql
```

**However, Go-based migrations are strongly recommended because:**
- Single codebase for all databases
- Type safety and compile-time checking
- Access to the full DAL
- Easier testing and debugging
- Better IDE support

To use SQL files:
```go
//go:embed migrations/*.sql
var migrationFiles embed.FS

source := migrations.NewEmbeddedSource("app", migrationFiles, "migrations", db)
```

## Troubleshooting

### Migrations Won't Run

Check for dirty database:
```go
statuses, _ := migrator.Status(ctx)
for _, s := range statuses {
    if s.Status == "failed" {
        fmt.Printf("Failed: %s - %s\n", s.Migration.FullName(), s.Error)
    }
}
```

### Checksum Mismatch

Migration modified after being applied. Options:

1. **Revert the change** (recommended)
2. **Update checksum** (if intentional):
   ```sql
   UPDATE schema_migrations
   SET checksum = 'new_checksum'
   WHERE version = '20251222204530123';
   ```

### Migration Fails

The migration runs in a transaction, so changes are rolled back. Fix and retry.

## Testing

Run the migration test suite:

```bash
cd migrations
go test -v -race

# With specific database
TEST_DATABASE_URL="postgres://localhost/test" go test -v

# Coverage
go test -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## License

Part of GoREST framework. See main LICENSE file.

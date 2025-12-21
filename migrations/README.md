# GoREST Database Migrations

A production-ready database migration system with support for multiple databases, plugin migrations, and comprehensive safety features.

## Features

- ✅ **Multi-Database Support**: PostgreSQL, MySQL, SQLite
- ✅ **Dialect-Specific Migrations**: Database-specific SQL files with generic fallback
- ✅ **Plugin System Integration**: Each plugin can maintain its own migrations
- ✅ **Checksum Verification**: Prevents migration drift between environments
- ✅ **Advisory Locking**: Prevents concurrent execution
- ✅ **Dirty Database Detection**: Tracks and blocks on failed migrations
- ✅ **Transaction Safety**: Automatic rollback on failure
- ✅ **Dependency Resolution**: Plugin migrations run in correct order

## Quick Start

### 1. Create Migration Files

Migration files use timestamp-based naming:

```
{timestamp}_{descriptive_name}.{up|down}[.{dialect}].sql
```

**Example:**
```
migrations/
├── 20250120143022_create_users.up.postgres.sql
├── 20250120143022_create_users.down.postgres.sql
├── 20250120143022_create_users.up.mysql.sql
├── 20250120143022_create_users.down.mysql.sql
├── 20250120143022_create_users.up.sqlite.sql
└── 20250120143022_create_users.down.sqlite.sql
```

**Generate Timestamp:**
```bash
date +%Y%m%d%H%M%S
# Output: 20250120143022
```

### 2. Embed Migrations in Your Application

```go
package main

import (
    "context"
    "embed"
    "log"

    "github.com/nicolasbonnici/gorest/database"
    "github.com/nicolasbonnici/gorest/migrations"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func main() {
    // Connect to database
    db, err := database.Open("postgres", "postgres://localhost/mydb")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // Create migration source
    appSource := migrations.NewEmbeddedSource("app", migrationFiles, "migrations", db)

    // Create migrator
    migrator := migrations.NewMigrator(db, appSource)

    // Run all pending migrations
    if err := migrator.Up(context.Background()); err != nil {
        log.Fatal(err)
    }

    log.Println("Migrations applied successfully")
}
```

### 3. Run Migrations

```go
ctx := context.Background()

// Apply all pending migrations
migrator.Up(ctx)

// Apply next pending migration
migrator.UpOne(ctx)

// Apply migrations up to specific version
migrator.UpTo(ctx, "20250120143022")

// Revert last migration
migrator.Down(ctx)

// Revert to specific version
migrator.DownTo(ctx, "20250120143022")

// Check migration status
statuses, _ := migrator.Status(ctx)
for _, s := range statuses {
    fmt.Printf("[%s] %s: %s\n", s.Migration.Source, s.Migration.FullName(), s.Status)
}
```

## Migration File Format

### Up Migration (PostgreSQL Example)

**20250120143022_create_users.up.postgres.sql:**
```sql
-- Create users table for authentication
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    firstname TEXT NOT NULL,
    lastname TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password TEXT,
    created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_email ON users (email);
```

### Down Migration

**20250120143022_create_users.down.postgres.sql:**
```sql
-- Rollback users table creation
DROP INDEX IF EXISTS idx_user_email;
DROP TABLE IF EXISTS users CASCADE;
```

### Dialect-Specific vs Generic Files

The migration system supports both dialect-specific and generic migration files:

1. **Dialect-Specific** (Recommended): Separate files per database
   - `20250120143022_create_users.up.postgres.sql`
   - `20250120143022_create_users.up.mysql.sql`
   - `20250120143022_create_users.up.sqlite.sql`

2. **Generic Fallback**: Single file for all databases
   - `20250120143022_create_users.up.sql`
   - Use only if SQL is truly database-agnostic

**Loading Priority:**
1. Try dialect-specific file first (e.g., `.postgres.sql`)
2. Fall back to generic file (e.g., `.sql`)

## Plugin Migrations

Plugins can provide their own migrations by implementing the `MigrationProvider` interface.

### Example: Auth Plugin

**plugins/auth/auth.go:**
```go
package auth

import (
    "embed"
    "github.com/nicolasbonnici/gorest/migrations"
    "github.com/nicolasbonnici/gorest/plugin"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type AuthPlugin struct {
    db database.Database
    // ...
}

// MigrationSource implements plugin.MigrationProvider
func (p *AuthPlugin) MigrationSource() plugin.MigrationSource {
    return migrations.NewEmbeddedSource("auth", migrationFiles, "migrations", p.db)
}

// MigrationDependencies returns dependencies
func (p *AuthPlugin) MigrationDependencies() []string {
    return []string{"app"} // Auth depends on app migrations
}
```

### Using Plugin Migrations

```go
package main

import (
    "github.com/nicolasbonnici/gorest/migrations"
    "github.com/nicolasbonnici/gorest/plugins/auth"
)

func main() {
    db, _ := database.Open("postgres", "postgres://localhost/mydb")
    defer db.Close()

    // Core app migrations
    appSource := migrations.NewEmbeddedSource("app", appMigrations, "migrations", db)

    // Collect plugin migrations
    sources := []migrations.MigrationSource{appSource}

    authPlugin := auth.NewPlugin()
    if provider, ok := authPlugin.(plugin.MigrationProvider); ok {
        sources = append(sources, provider.MigrationSource())
    }

    // Create migrator with all sources
    migrator := migrations.NewMigrator(db, sources...)

    // Set dependencies (if plugin implements MigrationDependencies)
    if provider, ok := authPlugin.(plugin.MigrationProvider); ok {
        deps := provider.MigrationDependencies()
        migrator.(*migrations.Migrator).SetSourceDependencies("auth", deps)
    }

    // Run all migrations (app + plugins)
    migrator.Up(context.Background())
}
```

## API Reference

### Migrator Interface

```go
type Migrator interface {
    // Apply all pending migrations
    Up(ctx context.Context) error

    // Apply migrations with custom options
    UpWithOptions(ctx context.Context, opts MigrationOptions) error

    // Apply next pending migration
    UpOne(ctx context.Context) error

    // Apply migrations up to specific version
    UpTo(ctx context.Context, version string) error

    // Revert last migration
    Down(ctx context.Context) error

    // Revert to specific version
    DownTo(ctx context.Context, version string) error

    // Get migration status
    Status(ctx context.Context) ([]MigrationStatus, error)

    // List pending migrations
    Pending(ctx context.Context) ([]Migration, error)

    // Validate migrations without executing
    Validate(ctx context.Context) error

    // Preview what would be executed
    DryRun(ctx context.Context) ([]Migration, error)

    // Force mark migration as applied (repair tool)
    Force(ctx context.Context, version, source string) error

    // Apply pending migrations for specific source
    UpSource(ctx context.Context, sourceName string) error

    // Revert last migration for specific source
    DownSource(ctx context.Context, sourceName string) error
}
```

### Migration Options

```go
type MigrationOptions struct {
    Transactional bool // Wrap all migrations in single transaction (all-or-nothing)
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

Every migration file is hashed (SHA256). If an applied migration file is modified, the system detects it and fails:

```
CRITICAL: Migration app/20250120143022_create_users has been modified after being applied!
Expected checksum: abc123...
Actual checksum:   def456...
DO NOT modify migrations that have been applied!
```

### 2. Advisory Locking

Prevents concurrent migrations from multiple deployments:

- **PostgreSQL**: Uses `pg_advisory_lock()`
- **MySQL**: Uses `GET_LOCK()` with 60s timeout
- **SQLite**: Uses EXCLUSIVE transactions

### 3. Dirty Database Detection

Tracks failed migrations and blocks execution until resolved:

```
Database is in dirty state - 1 failed migration(s) detected:
  - [app] 20250120143022_create_users: syntax error at line 5

You must:
  1. Fix the failing migration SQL and retry, OR
  2. Use Force() to mark as skipped (dangerous), OR
  3. Manually repair database and update schema_migrations status
```

### 4. Transaction Rollback

Each migration runs in its own transaction. If it fails:
- SQL changes are automatically rolled back
- Migration is marked as `status='failed'`
- Execution stops
- Database enters "dirty" state

### 5. Dependency Resolution

Plugin migrations respect dependencies via topological sort:

```go
// auth depends on app
func (p *AuthPlugin) MigrationDependencies() []string {
    return []string{"app"}
}

// Execution order: app → auth
```

## Schema Migrations Table

Migrations are tracked in the `schema_migrations` table:

```sql
CREATE TABLE schema_migrations (
    version VARCHAR(14) NOT NULL,             -- Timestamp: YYYYMMDDHHMMSS
    source VARCHAR(100) NOT NULL DEFAULT 'app',
    name VARCHAR(255) NOT NULL,
    checksum CHAR(64) NOT NULL,               -- SHA256 hash
    status VARCHAR(20) NOT NULL DEFAULT 'applied',
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    execution_time_ms INTEGER,
    executed_by VARCHAR(100),
    hostname VARCHAR(255),
    error_message TEXT,
    PRIMARY KEY (version, source)
);
```

**Query Migration Status:**
```sql
SELECT version, source, name, status, applied_at, execution_time_ms
FROM schema_migrations
ORDER BY applied_at DESC;
```

## Error Handling

### Common Errors

#### ErrDirtyDatabase
```go
if err == migrations.ErrDirtyDatabase {
    // Database has failed migrations
    // Fix SQL and retry, or use Force()
}
```

#### ErrChecksumMismatch
```go
if errors.Is(err, migrations.ErrChecksumMismatch) {
    // Migration file was modified after being applied
    // DO NOT modify applied migrations!
}
```

#### ErrLockTimeout
```go
if err == migrations.ErrLockTimeout {
    // Another process is running migrations
    // Wait and retry
}
```

### Recovery from Dirty Database

**Option 1: Fix SQL and Retry**
```bash
# Fix the failing migration file
# Then run migrations again
```

**Option 2: Force Migration (Dangerous)**
```go
// Mark migration as applied without executing
migrator.Force(ctx, "20250120143022", "app")
```

**Option 3: Manual Recovery**
```sql
-- Fix database manually
-- Then update migration status
UPDATE schema_migrations
SET status = 'applied', error_message = NULL
WHERE version = '20250120143022' AND source = 'app';
```

## Best Practices

### 1. Never Modify Applied Migrations

Once a migration is applied in any environment (dev, staging, prod), **never modify it**. Create a new migration instead.

❌ **Wrong:**
```sql
-- 20250120143022_create_users.up.sql (already applied)
CREATE TABLE users (
    id UUID PRIMARY KEY,
    name TEXT  -- Added column after migration was applied
);
```

✅ **Correct:**
```sql
-- 20250120150000_add_name_to_users.up.sql (new migration)
ALTER TABLE users ADD COLUMN name TEXT;
```

### 2. Always Provide Down Migrations

Every `up` migration must have a corresponding `down` migration that reverses it:

```sql
-- up: create table
CREATE TABLE users (id UUID PRIMARY KEY);

-- down: drop table
DROP TABLE IF EXISTS users;
```

### 3. Use Dialect-Specific Files

Different databases have different SQL syntax. Use dialect-specific files:

```
migrations/
├── 20250120143022_create_users.up.postgres.sql  (gen_random_uuid())
├── 20250120143022_create_users.up.mysql.sql     (UUID())
└── 20250120143022_create_users.up.sqlite.sql    (randomblob())
```

### 4. Test Migrations

Always test migrations:
- Test `up` migration
- Test `down` migration
- Test on all supported databases
- Test in a transaction

### 5. Keep Migrations Small

Each migration should do one logical thing:
- Create a table
- Add a column
- Create an index

Avoid:
- Multiple unrelated schema changes in one migration
- Large data migrations (use separate tool)

### 6. Use Descriptive Names

```
✅ 20250120143022_create_users_table.up.sql
✅ 20250120150000_add_email_index_to_users.up.sql
❌ 20250120143022_migration.up.sql
❌ 20250120143022_update.up.sql
```

## Troubleshooting

### Migrations Won't Run

**Check dirty database:**
```go
statuses, _ := migrator.Status(ctx)
for _, s := range statuses {
    if s.Status == "failed" {
        fmt.Printf("Failed: %s - %s\n", s.Migration.FullName(), s.Error)
    }
}
```

**Check for lock:**
```sql
-- PostgreSQL
SELECT * FROM pg_locks WHERE locktype = 'advisory';

-- MySQL
SELECT IS_USED_LOCK('gorest_migrations');
```

### Checksum Mismatch

Migration file was modified after being applied. Options:

1. **Revert the file change** (recommended)
2. **Update stored checksum** (if intentional):
   ```sql
   UPDATE schema_migrations
   SET checksum = 'new_checksum_here'
   WHERE version = '20250120143022' AND source = 'app';
   ```

### Migration Fails Mid-Execution

The migration is in a transaction, so changes are rolled back. Check:

1. SQL syntax errors
2. Constraint violations
3. Permission issues

Fix the SQL and re-run.

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

## Performance Tips

### 1. Add Indexes in Separate Migrations

```sql
-- Migration 1: Create table
CREATE TABLE users (id UUID PRIMARY KEY, email TEXT);

-- Migration 2: Add index (can be done online in production)
CREATE INDEX CONCURRENTLY idx_users_email ON users(email);
```

### 2. Use Migration Timeouts

```go
migration.Timeout = 5 * time.Minute  // For long-running migrations
```

### 3. Batch Large Data Migrations

For data migrations affecting millions of rows, use batching:

```sql
-- Instead of: UPDATE users SET status = 'active';
-- Use batching in application code or separate tool
```

## Examples

See:
- `/plugins/auth/migrations/` - Auth plugin migrations example
- `/migrations/testdata/` - Test migration examples
- `/examples/basic-api/` - Full application example

## License

Part of GoREST framework. See main LICENSE file.

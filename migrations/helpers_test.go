package migrations

import (
	"context"
	"testing"

	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
)

func TestGoMigration_Build(t *testing.T) {
	upCalled := false
	downCalled := false

	migration := NewGoMigration("20251222204530123", "test_migration").
		Up(func(ctx context.Context, db database.Database) error {
			upCalled = true
			return nil
		}).
		Down(func(ctx context.Context, db database.Database) error {
			downCalled = true
			return nil
		}).
		Build()

	if migration.Version != "20251222204530123" {
		t.Errorf("Expected version '20251222204530123', got '%s'", migration.Version)
	}

	if migration.Name != "test_migration" {
		t.Errorf("Expected name 'test_migration', got '%s'", migration.Name)
	}

	if migration.Executor == nil {
		t.Fatal("Executor should not be nil")
	}

	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Test Up
	if err := migration.ExecuteUp(ctx, db); err != nil {
		t.Fatalf("ExecuteUp failed: %v", err)
	}

	if !upCalled {
		t.Error("Up function was not called")
	}

	// Test Down
	if err := migration.ExecuteDown(ctx, db); err != nil {
		t.Fatalf("ExecuteDown failed: %v", err)
	}

	if !downCalled {
		t.Error("Down function was not called")
	}
}

func TestMigrationBuilder(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	builder := NewMigrationBuilder("test")

	builder.Add(
		"20251222204530123",
		"create_table",
		func(ctx context.Context, db database.Database) error {
			return CreateTableIfNotExists(ctx, db, "test_users", "id TEXT PRIMARY KEY, name TEXT")
		},
		func(ctx context.Context, db database.Database) error {
			return DropTableIfExists(ctx, db, "test_users")
		},
	)

	builder.Add(
		"20251222204530124",
		"add_index",
		func(ctx context.Context, db database.Database) error {
			return CreateIndex(ctx, db, "idx_test_users_name", "test_users", "name")
		},
		func(ctx context.Context, db database.Database) error {
			return DropIndex(ctx, db, "idx_test_users_name", "test_users")
		},
	)

	source := builder.Build()

	if source.Name() != "test" {
		t.Errorf("Expected source name 'test', got '%s'", source.Name())
	}

	migrations, err := source.Migrations()
	if err != nil {
		t.Fatalf("Failed to get migrations: %v", err)
	}

	if len(migrations) != 2 {
		t.Fatalf("Expected 2 migrations, got %d", len(migrations))
	}

	// Test migration execution
	migrator := NewMigrator(db, source)

	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify table was created
	_, err = db.Query(ctx, "SELECT id FROM test_users LIMIT 1")
	if err != nil {
		t.Errorf("Table should exist: %v", err)
	}

	// Test down migrations
	if err := migrator.Down(ctx); err != nil {
		t.Fatalf("Failed to revert migration: %v", err)
	}

	if err := migrator.Down(ctx); err != nil {
		t.Fatalf("Failed to revert migration: %v", err)
	}
}

func TestMigrationBuilder_AddSQL(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	builder := NewMigrationBuilder("test")

	builder.AddSQL(
		"20251222204530125",
		"create_sql_table",
		"CREATE TABLE sql_test (id INTEGER PRIMARY KEY)",
		"DROP TABLE IF EXISTS sql_test",
	)

	source := builder.Build()
	migrations, err := source.Migrations()
	if err != nil {
		t.Fatalf("Failed to get migrations: %v", err)
	}

	if len(migrations) != 1 {
		t.Fatalf("Expected 1 migration, got %d", len(migrations))
	}

	migrator := NewMigrator(db, source)

	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("Failed to run SQL migration: %v", err)
	}

	// Verify table exists
	_, err = db.Query(ctx, "SELECT id FROM sql_test LIMIT 1")
	if err != nil {
		t.Errorf("Table should exist: %v", err)
	}
}

func TestCreateTableIfNotExists(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	err := CreateTableIfNotExists(ctx, db, "helper_test", "id TEXT PRIMARY KEY, value TEXT")
	if err != nil {
		t.Fatalf("CreateTableIfNotExists failed: %v", err)
	}

	// Verify table exists
	_, err = db.Query(ctx, "SELECT id FROM helper_test LIMIT 1")
	if err != nil {
		t.Errorf("Table should exist: %v", err)
	}

	// Should be idempotent
	err = CreateTableIfNotExists(ctx, db, "helper_test", "id TEXT PRIMARY KEY, value TEXT")
	if err != nil {
		t.Errorf("CreateTableIfNotExists should be idempotent: %v", err)
	}
}

func TestDropTableIfExists(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Create table first
	err := CreateTableIfNotExists(ctx, db, "drop_test", "id TEXT PRIMARY KEY")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Drop it
	err = DropTableIfExists(ctx, db, "drop_test")
	if err != nil {
		t.Fatalf("DropTableIfExists failed: %v", err)
	}

	// Should be idempotent
	err = DropTableIfExists(ctx, db, "drop_test")
	if err != nil {
		t.Errorf("DropTableIfExists should be idempotent: %v", err)
	}
}

func TestAddColumn(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Create table first
	err := CreateTableIfNotExists(ctx, db, "column_test", "id TEXT PRIMARY KEY")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Add column
	err = AddColumn(ctx, db, "column_test", "name TEXT")
	if err != nil {
		t.Fatalf("AddColumn failed: %v", err)
	}

	// Verify column exists by inserting data
	_, err = db.Exec(ctx, "INSERT INTO column_test (id, name) VALUES (?, ?)", "1", "test")
	if err != nil {
		t.Errorf("Should be able to insert into new column: %v", err)
	}
}

func TestDropColumn(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Create table with two columns
	err := CreateTableIfNotExists(ctx, db, "drop_column_test", "id TEXT PRIMARY KEY, name TEXT, email TEXT")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Drop column (SQLite has limited support, skip if it fails)
	err = DropColumn(ctx, db, "drop_column_test", "email")
	if err != nil && db.DriverName() == "sqlite" {
		t.Skip("SQLite has limited DROP COLUMN support in older versions")
	} else if err != nil {
		t.Fatalf("DropColumn failed: %v", err)
	}
}

func TestCreateIndex(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Create table first
	err := CreateTableIfNotExists(ctx, db, "index_test", "id TEXT PRIMARY KEY, email TEXT")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create index
	err = CreateIndex(ctx, db, "idx_index_test_email", "index_test", "email")
	if err != nil {
		t.Fatalf("CreateIndex failed: %v", err)
	}

	// MySQL doesn't support IF NOT EXISTS, so we can't test idempotency there
	if db.DriverName() != "mysql" {
		err = CreateIndex(ctx, db, "idx_index_test_email", "index_test", "email")
		if err != nil {
			t.Errorf("CreateIndex should be idempotent: %v", err)
		}
	}
}

func TestDropIndex(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Create table and index first
	err := CreateTableIfNotExists(ctx, db, "drop_index_test", "id TEXT PRIMARY KEY, email TEXT")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	err = CreateIndex(ctx, db, "idx_drop_index_test_email", "drop_index_test", "email")
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}

	// Drop index
	err = DropIndex(ctx, db, "idx_drop_index_test_email", "drop_index_test")
	if err != nil {
		t.Fatalf("DropIndex failed: %v", err)
	}

	// Should be idempotent (except MySQL)
	if db.DriverName() != "mysql" {
		err = DropIndex(ctx, db, "idx_drop_index_test_email", "drop_index_test")
		if err != nil {
			t.Errorf("DropIndex should be idempotent: %v", err)
		}
	}
}

func TestDialectSQL(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	err := SQL(ctx, db, DialectSQL{
		Postgres: `CREATE TABLE dialect_test (
			id UUID PRIMARY KEY,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`,
		MySQL: `CREATE TABLE dialect_test (
			id CHAR(36) PRIMARY KEY,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		SQLite: `CREATE TABLE dialect_test (
			id TEXT PRIMARY KEY,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)`,
	})

	if err != nil {
		t.Fatalf("SQL failed: %v", err)
	}

	_, err = db.Query(ctx, "SELECT id FROM dialect_test LIMIT 1")
	if err != nil {
		t.Errorf("Table should exist: %v", err)
	}
}

func TestDialectSQL_OptionalFields(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	err := SQL(ctx, db, DialectSQL{
		SQLite: `CREATE TABLE optional_test (id TEXT PRIMARY KEY)`,
	})

	if err != nil {
		t.Fatalf("SQL with only SQLite field failed: %v", err)
	}

	_, err = db.Query(ctx, "SELECT id FROM optional_test LIMIT 1")
	if err != nil {
		t.Errorf("Table should exist: %v", err)
	}
}

func TestDialectSQL_MissingDialect(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	err := SQL(ctx, db, DialectSQL{
		Postgres: `CREATE TABLE missing_test (id UUID PRIMARY KEY)`,
		MySQL:    `CREATE TABLE missing_test (id CHAR(36) PRIMARY KEY)`,
	})

	if err == nil {
		t.Fatal("Expected error when SQL not provided for current dialect")
	}

	if err.Error() != "no SQL provided for sqlite dialect" {
		t.Errorf("Expected 'no SQL provided for sqlite dialect', got: %v", err)
	}
}

func TestGoMigrationExecutor_Checksum(t *testing.T) {
	upFunc := func(ctx context.Context, db database.Database) error {
		return nil
	}
	downFunc := func(ctx context.Context, db database.Database) error {
		return nil
	}

	executor1 := &GoMigrationExecutor{
		upFunc:   upFunc,
		downFunc: downFunc,
	}

	executor2 := &GoMigrationExecutor{
		upFunc:   upFunc,
		downFunc: downFunc,
	}

	// Same functions should produce same checksum
	checksum1 := executor1.Checksum()
	checksum2 := executor2.Checksum()

	if checksum1 == "" {
		t.Error("Checksum should not be empty")
	}

	if checksum1 != checksum2 {
		t.Error("Same functions should produce same checksum")
	}

	// Different functions should produce different checksum
	executor3 := &GoMigrationExecutor{
		upFunc: func(ctx context.Context, db database.Database) error {
			return nil
		},
		downFunc: downFunc,
	}

	checksum3 := executor3.Checksum()
	if checksum1 == checksum3 {
		t.Error("Different functions should produce different checksum")
	}
}

func TestGoMigrationExecutor_NoFunctions(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	executor := &GoMigrationExecutor{}

	// Should error when no up function
	err := executor.Up(ctx, db)
	if err == nil {
		t.Error("Expected error when no up function defined")
	}

	// Should error when no down function
	err = executor.Down(ctx, db)
	if err == nil {
		t.Error("Expected error when no down function defined")
	}
}

func TestIntegration_GoBasedMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)
	ctx := context.Background()

	// Build complete migration set
	builder := NewMigrationBuilder("integration_test")

	// Migration 1: Create users table
	builder.Add(
		"20251222204530123",
		"create_users_table",
		func(ctx context.Context, db database.Database) error {
			return CreateTableIfNotExists(ctx, db, "int_users",
				"id TEXT PRIMARY KEY, email TEXT UNIQUE NOT NULL, created_at TEXT")
		},
		func(ctx context.Context, db database.Database) error {
			return DropTableIfExists(ctx, db, "int_users")
		},
	)

	// Migration 2: Add index
	builder.Add(
		"20251222204530124",
		"add_users_email_index",
		func(ctx context.Context, db database.Database) error {
			return CreateIndex(ctx, db, "idx_int_users_email", "int_users", "email")
		},
		func(ctx context.Context, db database.Database) error {
			return DropIndex(ctx, db, "idx_int_users_email", "int_users")
		},
	)

	// Migration 3: Add column
	builder.Add(
		"20251222204530125",
		"add_users_name_column",
		func(ctx context.Context, db database.Database) error {
			return AddColumn(ctx, db, "int_users", "name TEXT")
		},
		func(ctx context.Context, db database.Database) error {
			return DropColumn(ctx, db, "int_users", "name")
		},
	)

	// Migration 4: Insert seed data
	builder.Add(
		"20251222204530126",
		"insert_admin_user",
		func(ctx context.Context, db database.Database) error {
			_, err := db.Exec(ctx,
				"INSERT INTO int_users (id, email, name) VALUES (?, ?, ?)",
				"admin-1", "admin@example.com", "Administrator")
			return err
		},
		func(ctx context.Context, db database.Database) error {
			_, err := db.Exec(ctx,
				"DELETE FROM int_users WHERE id = ?", "admin-1")
			return err
		},
	)

	source := builder.Build()
	migrator := NewMigrator(db, source)

	// Run all migrations
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify all migrations applied
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	for _, status := range statuses {
		if !status.Applied {
			t.Errorf("Migration %s should be applied", status.Migration.FullName())
		}
	}

	// Verify data
	var count int
	row := db.QueryRow(ctx, "SELECT COUNT(*) FROM int_users WHERE email = ?", "admin@example.com")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to query users: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 admin user, got %d", count)
	}

	// Rollback all migrations
	for i := 0; i < 4; i++ {
		if err := migrator.Down(ctx); err != nil {
			t.Fatalf("Failed to rollback migration %d: %v", i, err)
		}
	}

	// Verify all rolled back
	pending, err := migrator.Pending(ctx)
	if err != nil {
		t.Fatalf("Failed to get pending: %v", err)
	}

	if len(pending) != 4 {
		t.Errorf("Expected 4 pending migrations after rollback, got %d", len(pending))
	}
}

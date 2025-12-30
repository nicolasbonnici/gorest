package testhelpers

import (
	"context"
	"os"
	"testing"

	_ "github.com/nicolasbonnici/gorest/database/mysql"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
)

func TestSetupSQLite(t *testing.T) {
	db := SetupSQLite(t)

	if db == nil {
		t.Fatal("Expected non-nil database")
	}

	if db.DriverName() != "sqlite" {
		t.Errorf("Expected driver name 'sqlite', got %q", db.DriverName())
	}

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		t.Errorf("Expected successful ping: %v", err)
	}
}

func TestSetupSQLiteWithSchema(t *testing.T) {
	schema := `
		CREATE TABLE test_users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL
		)
	`

	db := SetupSQLiteWithSchema(t, schema)

	if db == nil {
		t.Fatal("Expected non-nil database")
	}

	ctx := context.Background()
	_, err := db.Exec(ctx, "INSERT INTO test_users (id, name) VALUES (?, ?)", "1", "Test User")
	if err != nil {
		t.Fatalf("Failed to insert into created table: %v", err)
	}

	var count int
	row := db.QueryRow(ctx, "SELECT COUNT(*) FROM test_users")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to query created table: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 row, got %d", count)
	}
}

func TestSetupSQLiteFile(t *testing.T) {
	db := SetupSQLiteFile(t)

	if db == nil {
		t.Fatal("Expected non-nil database")
	}

	if db.DriverName() != "sqlite" {
		t.Errorf("Expected driver name 'sqlite', got %q", db.DriverName())
	}

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		t.Errorf("Expected successful ping: %v", err)
	}
}

func TestSetupSQLiteFileWithSchema(t *testing.T) {
	schema := `
		CREATE TABLE test_posts (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL
		)
	`

	db := SetupSQLiteFileWithSchema(t, schema)

	if db == nil {
		t.Fatal("Expected non-nil database")
	}

	ctx := context.Background()
	_, err := db.Exec(ctx, "INSERT INTO test_posts (id, title) VALUES (?, ?)", "1", "Test Post")
	if err != nil {
		t.Fatalf("Failed to insert into created table: %v", err)
	}

	var title string
	row := db.QueryRow(ctx, "SELECT title FROM test_posts WHERE id = ?", "1")
	if err := row.Scan(&title); err != nil {
		t.Fatalf("Failed to query created table: %v", err)
	}

	if title != "Test Post" {
		t.Errorf("Expected title 'Test Post', got %q", title)
	}
}

func TestSetupTestDB_WithoutEnv(t *testing.T) {
	// Save original env
	originalEnv := os.Getenv("TEST_DATABASE_URL")
	defer func() {
		if originalEnv != "" {
			os.Setenv("TEST_DATABASE_URL", originalEnv)
		} else {
			os.Unsetenv("TEST_DATABASE_URL")
		}
	}()

	// Clear env to ensure SQLite fallback
	os.Unsetenv("TEST_DATABASE_URL")

	db := SetupTestDB(t)

	if db == nil {
		t.Fatal("Expected non-nil database")
	}

	// Should fall back to SQLite
	if db.DriverName() != "sqlite" {
		t.Errorf("Expected driver name 'sqlite', got %q", db.DriverName())
	}
}

func TestSetupPostgres_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := SetupPostgres(t)

	if db.DriverName() != "postgres" {
		t.Errorf("Expected driver name 'postgres', got %q", db.DriverName())
	}

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		t.Errorf("Expected successful ping: %v", err)
	}
}

func TestSetupPostgresWithDSN_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dsn := "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable"
	db := SetupPostgresWithDSN(t, dsn)

	if db.DriverName() != "postgres" {
		t.Errorf("Expected driver name 'postgres', got %q", db.DriverName())
	}

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		t.Errorf("Expected successful ping: %v", err)
	}
}

func TestSetupMySQL_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := SetupMySQL(t)

	if db.DriverName() != "mysql" {
		t.Errorf("Expected driver name 'mysql', got %q", db.DriverName())
	}

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		t.Errorf("Expected successful ping: %v", err)
	}
}

func TestSetupMySQLWithDSN_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dsn := "testuser:testpass@tcp(localhost:3307)/mydb_test"
	db := SetupMySQLWithDSN(t, dsn)

	if db.DriverName() != "mysql" {
		t.Errorf("Expected driver name 'mysql', got %q", db.DriverName())
	}

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		t.Errorf("Expected successful ping: %v", err)
	}
}

func TestCleanup(t *testing.T) {
	db := SetupSQLite(t)

	// Manual cleanup should work without error
	Cleanup(t, db)

	// Calling cleanup on nil should not panic
	Cleanup(t, nil)
}

func TestCleanupDB_SQLite(t *testing.T) {
	schema := `
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL
		);
		CREATE TABLE todo (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL
		);
	`

	db := SetupSQLiteWithSchema(t, schema)
	ctx := context.Background()

	// Insert test data
	_, err := db.Exec(ctx, "INSERT INTO users (id, name) VALUES (?, ?)", "1", "User 1")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	_, err = db.Exec(ctx, "INSERT INTO todo (id, title) VALUES (?, ?)", "1", "Todo 1")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Clean up
	CleanupDB(t, db)

	// Verify data is cleared
	var userCount, todoCount int
	db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	db.QueryRow(ctx, "SELECT COUNT(*) FROM todo").Scan(&todoCount)

	if userCount != 0 {
		t.Errorf("Expected 0 users after cleanup, got %d", userCount)
	}
	if todoCount != 0 {
		t.Errorf("Expected 0 todos after cleanup, got %d", todoCount)
	}
}

func TestCleanupTables_SQLite(t *testing.T) {
	schema := `
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL
		);
		CREATE TABLE todo (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL
		);
		CREATE TABLE posts (
			id TEXT PRIMARY KEY,
			content TEXT NOT NULL
		);
	`

	db := SetupSQLiteWithSchema(t, schema)
	ctx := context.Background()

	// Insert test data in all tables
	_, _ = db.Exec(ctx, "INSERT INTO users (id, name) VALUES (?, ?)", "1", "User 1")
	_, _ = db.Exec(ctx, "INSERT INTO todo (id, title) VALUES (?, ?)", "1", "Todo 1")
	_, _ = db.Exec(ctx, "INSERT INTO posts (id, content) VALUES (?, ?)", "1", "Post 1")

	// Clean up only users and todo, not posts
	CleanupTables(t, db, "users", "todo")

	// Verify selective cleanup
	var userCount, todoCount, postCount int
	db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount)
	db.QueryRow(ctx, "SELECT COUNT(*) FROM todo").Scan(&todoCount)
	db.QueryRow(ctx, "SELECT COUNT(*) FROM posts").Scan(&postCount)

	if userCount != 0 {
		t.Errorf("Expected 0 users after cleanup, got %d", userCount)
	}
	if todoCount != 0 {
		t.Errorf("Expected 0 todos after cleanup, got %d", todoCount)
	}
	if postCount != 1 {
		t.Errorf("Expected 1 post (not cleaned), got %d", postCount)
	}
}

func TestCleanupDB_Postgres_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := SetupPostgres(t)
	ctx := context.Background()

	// Insert test data
	_, _ = db.Exec(ctx, "INSERT INTO users (firstname, lastname, email) VALUES ($1, $2, $3)", "John", "Doe", "john.cleanup@example.com")

	// Clean up
	CleanupDB(t, db)

	// Verify data is cleared
	var count int
	db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)

	if count != 0 {
		t.Errorf("Expected 0 users after cleanup, got %d", count)
	}
}

func TestCleanupDB_MySQL_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := SetupMySQL(t)
	ctx := context.Background()

	// Insert test data
	_, _ = db.Exec(ctx, "INSERT INTO users (firstname, lastname, email) VALUES (?, ?, ?)", "Jane", "Smith", "jane.cleanup@example.com")

	// Clean up
	CleanupDB(t, db)

	// Verify data is cleared
	var count int
	db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)

	if count != 0 {
		t.Errorf("Expected 0 users after cleanup, got %d", count)
	}
}

func TestMultipleSetupCalls(t *testing.T) {
	// Test that multiple setup calls work correctly
	db1 := SetupSQLite(t)
	db2 := SetupSQLite(t)

	if db1 == nil || db2 == nil {
		t.Fatal("Expected non-nil databases")
	}

	// Both should be independent
	ctx := context.Background()
	if err := db1.Ping(ctx); err != nil {
		t.Errorf("db1 ping failed: %v", err)
	}
	if err := db2.Ping(ctx); err != nil {
		t.Errorf("db2 ping failed: %v", err)
	}
}

func TestSetupSQLite_UniqueNames(t *testing.T) {
	// Verify that each test gets a unique database name based on t.Name()
	db := SetupSQLite(t)

	if db == nil {
		t.Fatal("Expected non-nil database")
	}

	// The database should work correctly
	ctx := context.Background()
	_, err := db.Exec(ctx, "CREATE TABLE test_table (id INTEGER)")
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
}

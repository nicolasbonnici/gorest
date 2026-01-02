//go:build integration

package fixtures

import (
	"context"
	"fmt"
	"testing"

	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
)

func setupPostgresTestDB(t *testing.T) database.Database {
	t.Helper()

	// Use the docker-compose setup from test/compose.yml
	// PostgreSQL on port 5433
	dsn := "host=localhost port=5433 user=postgres password=postgres dbname=mydb_test sslmode=disable"

	db, err := database.Open("postgres", dsn)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v (run: docker-compose -f ../test/compose.yml up -d db_test)", err)
	}

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Skipf("PostgreSQL ping failed: %v (run: docker-compose -f ../test/compose.yml up -d db_test)", err)
	}

	// Drop and recreate tables
	schema := `
		DROP TABLE IF EXISTS todo CASCADE;
		DROP TABLE IF EXISTS users CASCADE;

		CREATE TABLE users (
			id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
			firstname TEXT NOT NULL,
			lastname TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT,
			updated_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE todo (
			id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
			user_id TEXT,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			updated_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`

	_, err = db.Exec(ctx, schema)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create schema: %v", err)
	}

	return db
}

// TestPostgreSQL_Placeholders tests that PostgreSQL placeholders ($1, $2, etc.) work correctly
func TestPostgreSQL_Placeholders(t *testing.T) {
	db := setupPostgresTestDB(t)
	defer db.Close()

	loader := New(db)
	users := []User{
		{ID: "pg-user-1", Firstname: "PostgreSQL", Lastname: "User1", Email: "pg1@test.com"},
		{ID: "pg-user-2", Firstname: "PostgreSQL", Lastname: "User2", Email: "pg2@test.com"},
		{ID: "pg-user-3", Firstname: "PostgreSQL", Lastname: "User3", Email: "pg3@test.com"},
	}

	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users with PostgreSQL placeholders: %v", err)
	}

	// Verify they were inserted
	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email LIKE 'pg%@test.com'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 users, got %d", count)
	}
}

// TestPostgreSQL_QuotedIdentifiers tests that identifier quoting works with PostgreSQL
func TestPostgreSQL_QuotedIdentifiers(t *testing.T) {
	db := setupPostgresTestDB(t)
	defer db.Close()

	loader := New(db)
	users := []User{
		{ID: "quoted-1", Firstname: "Quoted", Lastname: "Identifier", Email: "quoted@test.com"},
	}

	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users with quoted identifiers: %v", err)
	}

	// Verify the data
	ctx := context.Background()
	var email string
	err = db.QueryRow(ctx, `SELECT "email" FROM "users" WHERE "id" = $1`, "quoted-1").Scan(&email)
	if err != nil {
		t.Fatalf("failed to query with quoted identifiers: %v", err)
	}

	if email != "quoted@test.com" {
		t.Errorf("expected quoted@test.com, got %s", email)
	}
}

// TestPostgreSQL_TransactionRollback tests transaction rollback with PostgreSQL
func TestPostgreSQL_TransactionRollback(t *testing.T) {
	db := setupPostgresTestDB(t)
	defer db.Close()

	loader := New(db)
	_, err := loader.WithTransaction()
	if err != nil {
		t.Fatalf("failed to start transaction: %v", err)
	}

	users := []User{
		{ID: "tx-rollback-1", Firstname: "Transaction", Lastname: "Rollback", Email: "txrollback@test.com"},
	}

	_, err = Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	err = loader.Rollback()
	if err != nil {
		t.Fatalf("failed to rollback: %v", err)
	}

	// Verify data was rolled back
	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = 'txrollback@test.com'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 users after rollback, got %d", count)
	}
}

// TestPostgreSQL_ForeignKeyCleanup tests cleanup with foreign key constraints
func TestPostgreSQL_ForeignKeyCleanup(t *testing.T) {
	db := setupPostgresTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()

	users := []User{
		{ID: "fk-user-1", Firstname: "FK", Lastname: "User", Email: "fkuser@test.com"},
	}
	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	todos := []Todo{
		{ID: "fk-todo-1", UserID: "fk-user-1", Title: "FK Todo", Content: "Content"},
	}
	_, err = Load(loader, "todos", todos)
	if err != nil {
		t.Fatalf("failed to load todos: %v", err)
	}

	// Cleanup in correct order (child before parent)
	err = CleanupOrdered(loader, []string{"users", "todos"})
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// Verify both tables are empty
	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 users, got %d", count)
	}

	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM todo").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count todos: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 todos, got %d", count)
	}
}

// TestPostgreSQL_DeleteWithPlaceholders tests DELETE queries with PostgreSQL placeholders
func TestPostgreSQL_DeleteWithPlaceholders(t *testing.T) {
	db := setupPostgresTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()

	users := []User{
		{ID: "del-1", Firstname: "Delete", Lastname: "Test1", Email: "del1@test.com"},
		{ID: "del-2", Firstname: "Delete", Lastname: "Test2", Email: "del2@test.com"},
		{ID: "del-3", Firstname: "Delete", Lastname: "Test3", Email: "del3@test.com"},
	}

	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	// Cleanup should use $1, $2, $3 placeholders
	err = Cleanup(loader)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// Verify all deleted
	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email LIKE 'del%@test.com'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 users after cleanup, got %d", count)
	}
}

// TestPostgreSQL_Builder tests the builder pattern with PostgreSQL
func TestPostgreSQL_Builder(t *testing.T) {
	db := setupPostgresTestDB(t)
	defer db.Close()

	users := []User{
		{ID: "builder-pg-1", Firstname: "Builder", Lastname: "PostgreSQL", Email: "builderpg@test.com"},
	}

	builder := NewBuilderWithT(t, db)
	LoadBuilder(builder.WithTransaction(), "users", users)
	builder.Commit()

	if builder.Error() != nil {
		t.Fatalf("unexpected error: %v", builder.Error())
	}

	// Verify the data
	ctx := context.Background()
	var count int
	err := db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = 'builderpg@test.com'").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}
}

// TestPostgreSQL_ConcurrentLoad tests thread-safe loading with PostgreSQL
func TestPostgreSQL_ConcurrentLoad(t *testing.T) {
	db := setupPostgresTestDB(t)
	defer db.Close()

	loader := New(db)

	// Load fixtures concurrently
	done := make(chan bool, 2)

	go func() {
		users := []User{
			{ID: fmt.Sprintf("concurrent-user-%d", 1), Firstname: "Concurrent", Lastname: "User1", Email: fmt.Sprintf("concurrent1@test.com")},
		}
		_, err := Load(loader, "users1", users)
		if err != nil {
			t.Errorf("failed to load users1: %v", err)
		}
		done <- true
	}()

	go func() {
		users := []User{
			{ID: fmt.Sprintf("concurrent-user-%d", 2), Firstname: "Concurrent", Lastname: "User2", Email: fmt.Sprintf("concurrent2@test.com")},
		}
		_, err := Load(loader, "users2", users)
		if err != nil {
			t.Errorf("failed to load users2: %v", err)
		}
		done <- true
	}()

	<-done
	<-done

	// Verify both were loaded
	fixtures1, ok1 := loader.Get("users1")
	fixtures2, ok2 := loader.Get("users2")

	if !ok1 || !ok2 {
		t.Error("expected both fixture sets to be loaded")
	}

	if len(fixtures1) != 1 || len(fixtures2) != 1 {
		t.Errorf("expected 1 fixture in each set, got %d and %d", len(fixtures1), len(fixtures2))
	}
}

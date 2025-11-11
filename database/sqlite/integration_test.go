//go:build integration

package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/nicolasbonnici/gorest/database"
)

func setupSQLiteTest(t *testing.T) database.Database {
	t.Helper()

	db, err := database.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Fatalf("SQLite ping failed: %v", err)
	}

	schemaPath := findSchemaFile(t)
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to read schema: %v", err)
	}

	_, err = db.Exec(ctx, string(schema))
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

func findSchemaFile(t *testing.T) string {
	t.Helper()

	paths := []string{
		"../../../test/sql/schema_sqlite.sql",
		"../../test/sql/schema_sqlite.sql",
		"test/sql/schema_sqlite.sql",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			absPath, _ := filepath.Abs(path)
			return absPath
		}
	}

	t.Fatal("Could not find schema_sqlite.sql")
	return ""
}

func cleanupSQLite(t *testing.T, db database.Database) {
	t.Helper()
	ctx := context.Background()

	_, _ = db.Exec(ctx, "DELETE FROM todo")
	_, _ = db.Exec(ctx, "DELETE FROM users")
}

func TestSQLite_Connection(t *testing.T) {
	db := setupSQLiteTest(t)
	defer db.Close()

	if db.DriverName() != "sqlite" {
		t.Errorf("Expected driver name 'sqlite', got %q", db.DriverName())
	}
}

func TestSQLite_Dialect(t *testing.T) {
	db := setupSQLiteTest(t)
	defer db.Close()

	dialect := db.Dialect()

	if dialect.Placeholder(1) != "?" {
		t.Errorf("Expected ? placeholder, got %q", dialect.Placeholder(1))
	}

	if !dialect.SupportsReturning() {
		t.Error("SQLite should support RETURNING")
	}

	if dialect.QuoteIdentifier("table") != `"table"` {
		t.Errorf("Expected double quote quoting, got %q", dialect.QuoteIdentifier("table"))
	}
}

func TestSQLite_Insert(t *testing.T) {
	db := setupSQLiteTest(t)
	defer db.Close()

	ctx := context.Background()

	query := "INSERT INTO users (firstname, lastname, email, password) VALUES (?, ?, ?, ?)"
	res, err := db.Exec(ctx, query, "John", "Doe", "john@example.com", "pass123")
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get last insert ID: %v", err)
	}

	if id == 0 {
		t.Error("Expected non-zero insert ID")
	}

	affected, err := res.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}

	if affected != 1 {
		t.Errorf("Expected 1 row affected, got %d", affected)
	}
}

func TestSQLite_InsertWithReturning(t *testing.T) {
	db := setupSQLiteTest(t)
	defer db.Close()

	ctx := context.Background()

	query := "INSERT INTO users (firstname, lastname, email, password) VALUES (?, ?, ?, ?) RETURNING id"
	var id string
	row := db.QueryRow(ctx, query, "Alice", "Smith", "alice@example.com", "pass123")
	if err := row.Scan(&id); err != nil {
		t.Fatalf("Failed to scan returned ID: %v", err)
	}

	if id == "" {
		t.Error("Expected non-empty ID")
	}
}

func TestSQLite_Query(t *testing.T) {
	db := setupSQLiteTest(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(ctx, "INSERT INTO users (firstname, lastname, email) VALUES (?, ?, ?)", "Bob", "Jones", "bob@example.com")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	rows, err := db.Query(ctx, "SELECT firstname, lastname, email FROM users WHERE email = ?", "bob@example.com")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatal("Expected at least one row")
	}

	var firstname, lastname, email string
	if err := rows.Scan(&firstname, &lastname, &email); err != nil {
		t.Fatalf("Failed to scan: %v", err)
	}

	if firstname != "Bob" || lastname != "Jones" || email != "bob@example.com" {
		t.Errorf("Unexpected values: %s %s %s", firstname, lastname, email)
	}
}

func TestSQLite_QueryRow(t *testing.T) {
	db := setupSQLiteTest(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(ctx, "INSERT INTO users (firstname, lastname, email) VALUES (?, ?, ?)", "Charlie", "Brown", "charlie@example.com")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	var count int
	row := db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = ?", "charlie@example.com")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to scan: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 user, got %d", count)
	}
}

func TestSQLite_Update(t *testing.T) {
	db := setupSQLiteTest(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(ctx, "INSERT INTO users (firstname, lastname, email) VALUES (?, ?, ?)", "David", "Miller", "david@example.com")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	res, err := db.Exec(ctx, "UPDATE users SET lastname = ? WHERE email = ?", "Wilson", "david@example.com")
	if err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}

	if affected != 1 {
		t.Errorf("Expected 1 row affected, got %d", affected)
	}

	var lastname string
	row := db.QueryRow(ctx, "SELECT lastname FROM users WHERE email = ?", "david@example.com")
	if err := row.Scan(&lastname); err != nil {
		t.Fatalf("Failed to scan: %v", err)
	}

	if lastname != "Wilson" {
		t.Errorf("Expected lastname 'Wilson', got %q", lastname)
	}
}

func TestSQLite_Delete(t *testing.T) {
	db := setupSQLiteTest(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(ctx, "INSERT INTO users (firstname, lastname, email) VALUES (?, ?, ?)", "Emma", "Taylor", "emma@example.com")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	res, err := db.Exec(ctx, "DELETE FROM users WHERE email = ?", "emma@example.com")
	if err != nil {
		t.Fatalf("Failed to delete: %v", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}

	if affected != 1 {
		t.Errorf("Expected 1 row affected, got %d", affected)
	}
}

func TestSQLite_SchemaIntrospection(t *testing.T) {
	db := setupSQLiteTest(t)
	defer db.Close()

	ctx := context.Background()

	schema, err := db.Introspector().LoadSchema(ctx)
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	if len(schema) < 2 {
		t.Errorf("Expected at least 2 tables, got %d", len(schema))
	}

	var foundUsers, foundTodo bool
	for _, table := range schema {
		if table.TableName == "users" {
			foundUsers = true
			if len(table.Columns) < 5 {
				t.Errorf("Expected at least 5 columns in users table, got %d", len(table.Columns))
			}
		}
		if table.TableName == "todo" {
			foundTodo = true
			if len(table.Columns) < 5 {
				t.Errorf("Expected at least 5 columns in todo table, got %d", len(table.Columns))
			}
		}
	}

	if !foundUsers {
		t.Error("users table not found in schema")
	}
	if !foundTodo {
		t.Error("todo table not found in schema")
	}
}

func TestSQLite_ForeignKey(t *testing.T) {
	db := setupSQLiteTest(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(ctx, "PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	var userId string
	row := db.QueryRow(ctx, "INSERT INTO users (firstname, lastname, email) VALUES (?, ?, ?) RETURNING id", "Test", "User", "test@example.com")
	if err := row.Scan(&userId); err != nil {
		t.Fatalf("Failed to insert user: %v", err)
	}

	_, err = db.Exec(ctx, "INSERT INTO todo (user_id, title, content) VALUES (?, ?, ?)", userId, "Test Todo", "Content")
	if err != nil {
		t.Fatalf("Failed to insert todo with valid foreign key: %v", err)
	}

	_, err = db.Exec(ctx, "INSERT INTO todo (user_id, title, content) VALUES (?, ?, ?)", "invalid-id", "Bad Todo", "Content")
	if err == nil {
		t.Error("Expected foreign key constraint error, got none")
	}
}

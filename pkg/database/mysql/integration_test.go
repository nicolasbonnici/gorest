//go:build integration

package mysql

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nicolasbonnici/gorest/pkg/database"
)

const defaultMySQLTestURL = "testuser:testpass@tcp(localhost:3307)/mydb_test"

func setupMySQLTest(t *testing.T) database.Database {
	t.Helper()

	testURL := os.Getenv("MYSQL_TEST_URL")
	if testURL == "" {
		testURL = defaultMySQLTestURL
	}

	db, err := database.Open("mysql", testURL)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Skipf("MySQL ping failed: %v", err)
	}

	cleanupMySQL(t, db)
	return db
}

func cleanupMySQL(t *testing.T, db database.Database) {
	t.Helper()
	ctx := context.Background()

	_, _ = db.Exec(ctx, "SET FOREIGN_KEY_CHECKS = 0")
	_, _ = db.Exec(ctx, "TRUNCATE TABLE todo")
	_, _ = db.Exec(ctx, "TRUNCATE TABLE users")
	_, _ = db.Exec(ctx, "SET FOREIGN_KEY_CHECKS = 1")
}

func TestMySQL_Connection(t *testing.T) {
	db := setupMySQLTest(t)
	defer db.Close()

	if db.DriverName() != "mysql" {
		t.Errorf("Expected driver name 'mysql', got %q", db.DriverName())
	}
}

func TestMySQL_Dialect(t *testing.T) {
	db := setupMySQLTest(t)
	defer db.Close()

	dialect := db.Dialect()

	if dialect.Placeholder(1) != "?" {
		t.Errorf("Expected ? placeholder, got %q", dialect.Placeholder(1))
	}

	if dialect.SupportsReturning() {
		t.Error("MySQL should not support RETURNING")
	}

	if dialect.QuoteIdentifier("table") != "`table`" {
		t.Errorf("Expected backtick quoting, got %q", dialect.QuoteIdentifier("table"))
	}
}

func TestMySQL_Insert(t *testing.T) {
	db := setupMySQLTest(t)
	defer db.Close()

	ctx := context.Background()

	query := "INSERT INTO users (firstname, lastname, email, password) VALUES (?, ?, ?, ?)"
	res, err := db.Exec(ctx, query, "John", "Doe", "john@example.com", "pass123")
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// MySQL UUID() doesn't provide an integer LastInsertId, so we skip that check
	// Just verify the insert worked by checking rows affected
	affected, err := res.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}

	if affected != 1 {
		t.Errorf("Expected 1 row affected, got %d", affected)
	}

	// Verify the insert by querying for the inserted record
	var count int
	row := db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = ?", "john@example.com")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to verify insert: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 inserted row, got %d", count)
	}
}

func TestMySQL_Query(t *testing.T) {
	db := setupMySQLTest(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(ctx, "INSERT INTO users (firstname, lastname, email) VALUES (?, ?, ?)", "Alice", "Smith", "alice@example.com")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	rows, err := db.Query(ctx, "SELECT firstname, lastname, email FROM users WHERE email = ?", "alice@example.com")
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

	if firstname != "Alice" || lastname != "Smith" || email != "alice@example.com" {
		t.Errorf("Unexpected values: %s %s %s", firstname, lastname, email)
	}
}

func TestMySQL_QueryRow(t *testing.T) {
	db := setupMySQLTest(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(ctx, "INSERT INTO users (firstname, lastname, email) VALUES (?, ?, ?)", "Bob", "Jones", "bob@example.com")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	var count int
	row := db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = ?", "bob@example.com")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to scan: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 user, got %d", count)
	}
}

func TestMySQL_Update(t *testing.T) {
	db := setupMySQLTest(t)
	defer db.Close()

	ctx := context.Background()

	insertRes, err := db.Exec(ctx, "INSERT INTO users (firstname, lastname, email) VALUES (?, ?, ?)", "Charlie", "Brown", "charlie@example.com")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Verify insert succeeded
	insertAffected, _ := insertRes.RowsAffected()
	if insertAffected != 1 {
		t.Fatalf("Insert failed, expected 1 row affected, got %d", insertAffected)
	}

	// Verify the row exists before update
	var countBefore int
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = ?", "charlie@example.com").Scan(&countBefore); err != nil {
		t.Fatalf("Failed to verify insert: %v", err)
	}
	if countBefore != 1 {
		t.Fatalf("Row not found after insert, count: %d", countBefore)
	}

	res, err := db.Exec(ctx, "UPDATE users SET lastname = ? WHERE email = ?", "Wilson", "charlie@example.com")
	if err != nil {
		t.Fatalf("Failed to update: %v", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		t.Fatalf("Failed to get rows affected: %v", err)
	}

	if affected != 1 {
		t.Errorf("Expected 1 row affected by update, got %d", affected)
	}

	var lastname string
	row := db.QueryRow(ctx, "SELECT lastname FROM users WHERE email = ?", "charlie@example.com")
	if err := row.Scan(&lastname); err != nil {
		t.Fatalf("Failed to scan lastname: %v", err)
	}

	if lastname != "Wilson" {
		t.Errorf("Expected lastname 'Wilson', got %q", lastname)
	}
}

func TestMySQL_Delete(t *testing.T) {
	db := setupMySQLTest(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(ctx, "INSERT INTO users (firstname, lastname, email) VALUES (?, ?, ?)", "David", "Miller", "david@example.com")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	res, err := db.Exec(ctx, "DELETE FROM users WHERE email = ?", "david@example.com")
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

func TestMySQL_SchemaIntrospection(t *testing.T) {
	db := setupMySQLTest(t)
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

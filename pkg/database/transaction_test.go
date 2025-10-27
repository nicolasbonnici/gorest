//go:build integration

package database_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/nicolasbonnici/gorest/pkg/database"
	_ "github.com/nicolasbonnici/gorest/pkg/database/mysql"
	_ "github.com/nicolasbonnici/gorest/pkg/database/postgres"
	_ "github.com/nicolasbonnici/gorest/pkg/database/sqlite"
)

func TestTransaction_CommitPostgreSQL(t *testing.T) {
	db := setupTestDB(t, "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable")
	defer db.Close()
	testTransactionCommit(t, db)
}

func TestTransaction_CommitMySQL(t *testing.T) {
	db := setupTestDB(t, "testuser:testpass@tcp(localhost:3307)/mydb_test")
	defer db.Close()
	testTransactionCommit(t, db)
}

func TestTransaction_CommitSQLite(t *testing.T) {
	db := setupSQLiteTestDB(t)
	defer db.Close()
	testTransactionCommit(t, db)
}

func testTransactionCommit(t *testing.T, db database.Database) {
	ctx := context.Background()
	cleanupDB(t, db)

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	query := "INSERT INTO users (firstname, lastname, email) VALUES (" +
		db.Dialect().Placeholder(1) + ", " +
		db.Dialect().Placeholder(2) + ", " +
		db.Dialect().Placeholder(3) + ")"

	_, err = tx.Exec(ctx, query, "John", "Doe", "john.tx@example.com")
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	var count int
	countQuery := "SELECT COUNT(*) FROM users WHERE email = " + db.Dialect().Placeholder(1)
	row := db.QueryRow(ctx, countQuery, "john.tx@example.com")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to verify commit: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 user after commit, got %d", count)
	}
}

func TestTransaction_RollbackPostgreSQL(t *testing.T) {
	db := setupTestDB(t, "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable")
	defer db.Close()
	testTransactionRollback(t, db)
}

func TestTransaction_RollbackMySQL(t *testing.T) {
	db := setupTestDB(t, "testuser:testpass@tcp(localhost:3307)/mydb_test")
	defer db.Close()
	testTransactionRollback(t, db)
}

func TestTransaction_RollbackSQLite(t *testing.T) {
	db := setupSQLiteTestDB(t)
	defer db.Close()
	testTransactionRollback(t, db)
}

func testTransactionRollback(t *testing.T, db database.Database) {
	ctx := context.Background()
	cleanupDB(t, db)

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	query := "INSERT INTO users (firstname, lastname, email) VALUES (" +
		db.Dialect().Placeholder(1) + ", " +
		db.Dialect().Placeholder(2) + ", " +
		db.Dialect().Placeholder(3) + ")"

	_, err = tx.Exec(ctx, query, "Alice", "Smith", "alice.rollback@example.com")
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("Failed to insert in transaction: %v", err)
	}

	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("Failed to rollback transaction: %v", err)
	}

	var count int
	countQuery := "SELECT COUNT(*) FROM users WHERE email = " + db.Dialect().Placeholder(1)
	row := db.QueryRow(ctx, countQuery, "alice.rollback@example.com")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to verify rollback: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 users after rollback, got %d", count)
	}
}

func TestTransaction_RollbackOnErrorPostgreSQL(t *testing.T) {
	db := setupTestDB(t, "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable")
	defer db.Close()
	testTransactionRollbackOnError(t, db)
}

func TestTransaction_RollbackOnErrorMySQL(t *testing.T) {
	db := setupTestDB(t, "testuser:testpass@tcp(localhost:3307)/mydb_test")
	defer db.Close()
	testTransactionRollbackOnError(t, db)
}

func TestTransaction_RollbackOnErrorSQLite(t *testing.T) {
	db := setupSQLiteTestDB(t)
	defer db.Close()
	testTransactionRollbackOnError(t, db)
}

func testTransactionRollbackOnError(t *testing.T, db database.Database) {
	ctx := context.Background()
	cleanupDB(t, db)

	query := "INSERT INTO users (firstname, lastname, email) VALUES (" +
		db.Dialect().Placeholder(1) + ", " +
		db.Dialect().Placeholder(2) + ", " +
		db.Dialect().Placeholder(3) + ")"

	_, err := db.Exec(ctx, query, "Bob", "Jones", "bob.unique@example.com")
	if err != nil {
		t.Fatalf("Failed to insert initial user: %v", err)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	_, err = tx.Exec(ctx, query, "Bob", "Jones", "bob.unique@example.com")
	if err == nil {
		tx.Rollback(ctx)
		t.Fatal("Expected unique constraint error, got none")
	}

	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("Failed to rollback after error: %v", err)
	}

	var count int
	countQuery := "SELECT COUNT(*) FROM users WHERE email = " + db.Dialect().Placeholder(1)
	row := db.QueryRow(ctx, countQuery, "bob.unique@example.com")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to verify: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 user (original), got %d", count)
	}
}

func TestTransaction_MultipleOperationsPostgreSQL(t *testing.T) {
	db := setupTestDB(t, "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable")
	defer db.Close()
	testTransactionMultipleOperations(t, db)
}

func TestTransaction_MultipleOperationsMySQL(t *testing.T) {
	db := setupTestDB(t, "testuser:testpass@tcp(localhost:3307)/mydb_test")
	defer db.Close()
	testTransactionMultipleOperations(t, db)
}

func TestTransaction_MultipleOperationsSQLite(t *testing.T) {
	db := setupSQLiteTestDB(t)
	defer db.Close()
	testTransactionMultipleOperations(t, db)
}

func testTransactionMultipleOperations(t *testing.T, db database.Database) {
	ctx := context.Background()
	cleanupDB(t, db)

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	userQuery := "INSERT INTO users (firstname, lastname, email) VALUES (" +
		db.Dialect().Placeholder(1) + ", " +
		db.Dialect().Placeholder(2) + ", " +
		db.Dialect().Placeholder(3) + ")"

	var userID string
	if db.Dialect().SupportsReturning() {
		row := tx.QueryRow(ctx, userQuery+" "+db.Dialect().ReturningClause(), "Charlie", "Brown", "charlie.multi@example.com")
		if err := row.Scan(&userID); err != nil {
			tx.Rollback(ctx)
			t.Fatalf("Failed to insert user: %v", err)
		}
	} else {
		res, err := tx.Exec(ctx, userQuery, "Charlie", "Brown", "charlie.multi@example.com")
		if err != nil {
			tx.Rollback(ctx)
			t.Fatalf("Failed to insert user: %v", err)
		}
		id, _ := res.LastInsertId()
		userID = string(rune(id))
	}

	updateQuery := "UPDATE users SET lastname = " + db.Dialect().Placeholder(1) +
		" WHERE email = " + db.Dialect().Placeholder(2)
	_, err = tx.Exec(ctx, updateQuery, "Wilson", "charlie.multi@example.com")
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("Failed to update user: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}

	var lastname string
	selectQuery := "SELECT lastname FROM users WHERE email = " + db.Dialect().Placeholder(1)
	row := db.QueryRow(ctx, selectQuery, "charlie.multi@example.com")
	if err := row.Scan(&lastname); err != nil {
		t.Fatalf("Failed to verify: %v", err)
	}

	if lastname != "Wilson" {
		t.Errorf("Expected lastname 'Wilson', got %q", lastname)
	}
}

func setupTestDB(t *testing.T, dsn string) database.Database {
	t.Helper()

	db, err := database.Open("", dsn)
	if err != nil {
		t.Skipf("Database not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Skipf("Database ping failed: %v", err)
	}

	cleanupDB(t, db)
	return db
}

func setupSQLiteTestDB(t *testing.T) database.Database {
	t.Helper()

	// Use file-based database with a unique name for this test to avoid concurrent access issues
	// SQLite :memory: databases are connection-specific, causing failures in concurrent tests
	dbPath := fmt.Sprintf("/tmp/gorest_test_%s.db", t.Name())

	// Clean up any existing database
	_ = os.Remove(dbPath)

	db, err := database.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}

	// Clean up database file when test completes
	t.Cleanup(func() {
		db.Close()
		_ = os.Remove(dbPath)
	})

	ctx := context.Background()
	schema := `
CREATE TABLE users (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    firstname TEXT NOT NULL,
    lastname TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password TEXT,
    updated_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE todo (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    user_id TEXT,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    updated_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);`

	_, err = db.Exec(ctx, schema)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Enable WAL mode for better concurrency and set busy timeout
	_, _ = db.Exec(ctx, "PRAGMA journal_mode=WAL")
	_, _ = db.Exec(ctx, "PRAGMA busy_timeout=5000")

	return db
}

func cleanupDB(t *testing.T, db database.Database) {
	t.Helper()
	ctx := context.Background()

	switch db.DriverName() {
	case "postgres":
		_, _ = db.Exec(ctx, "TRUNCATE users, todo CASCADE")
	case "mysql":
		_, _ = db.Exec(ctx, "SET FOREIGN_KEY_CHECKS = 0")
		_, _ = db.Exec(ctx, "TRUNCATE users")
		_, _ = db.Exec(ctx, "TRUNCATE todo")
		_, _ = db.Exec(ctx, "SET FOREIGN_KEY_CHECKS = 1")
	case "sqlite":
		_, _ = db.Exec(ctx, "DELETE FROM todo")
		_, _ = db.Exec(ctx, "DELETE FROM users")
	}
}

func TestTransaction_QueryPostgreSQL(t *testing.T) {
	db := setupTestDB(t, "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable")
	defer db.Close()
	testTransactionQuery(t, db)
}

func TestTransaction_QueryMySQL(t *testing.T) {
	db := setupTestDB(t, "testuser:testpass@tcp(localhost:3307)/mydb_test")
	defer db.Close()
	testTransactionQuery(t, db)
}

func TestTransaction_QuerySQLite(t *testing.T) {
	db := setupSQLiteTestDB(t)
	defer db.Close()
	testTransactionQuery(t, db)
}

func testTransactionQuery(t *testing.T, db database.Database) {
	ctx := context.Background()
	cleanupDB(t, db)

	// Insert test data
	insertQuery := "INSERT INTO users (firstname, lastname, email) VALUES (" +
		db.Dialect().Placeholder(1) + ", " +
		db.Dialect().Placeholder(2) + ", " +
		db.Dialect().Placeholder(3) + ")"
	_, err := db.Exec(ctx, insertQuery, "Alice", "Query", "alice.query@example.com")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}
	_, err = db.Exec(ctx, insertQuery, "Bob", "Query", "bob.query@example.com")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Begin transaction and use Query
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	query := "SELECT firstname, email FROM users WHERE lastname = " + db.Dialect().Placeholder(1) + " ORDER BY email"
	rows, err := tx.Query(ctx, query, "Query")
	if err != nil {
		t.Fatalf("Transaction Query failed: %v", err)
	}
	defer rows.Close()

	var count int
	for rows.Next() {
		var firstname, email string
		if err := rows.Scan(&firstname, &email); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		count++
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("Rows iteration error: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 rows, got %d", count)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
}

func TestTransaction_QueryRowPostgreSQL(t *testing.T) {
	db := setupTestDB(t, "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable")
	defer db.Close()
	testTransactionQueryRow(t, db)
}

func TestTransaction_QueryRowMySQL(t *testing.T) {
	db := setupTestDB(t, "testuser:testpass@tcp(localhost:3307)/mydb_test")
	defer db.Close()
	testTransactionQueryRow(t, db)
}

func TestTransaction_QueryRowSQLite(t *testing.T) {
	db := setupSQLiteTestDB(t)
	defer db.Close()
	testTransactionQueryRow(t, db)
}

func testTransactionQueryRow(t *testing.T, db database.Database) {
	ctx := context.Background()
	cleanupDB(t, db)

	// Insert test data
	insertQuery := "INSERT INTO users (firstname, lastname, email) VALUES (" +
		db.Dialect().Placeholder(1) + ", " +
		db.Dialect().Placeholder(2) + ", " +
		db.Dialect().Placeholder(3) + ")"
	_, err := db.Exec(ctx, insertQuery, "Charlie", "QueryRow", "charlie.queryrow@example.com")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Begin transaction and use QueryRow
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	query := "SELECT firstname, lastname FROM users WHERE email = " + db.Dialect().Placeholder(1)
	row := tx.QueryRow(ctx, query, "charlie.queryrow@example.com")

	var firstname, lastname string
	if err := row.Scan(&firstname, &lastname); err != nil {
		t.Fatalf("QueryRow Scan failed: %v", err)
	}

	if firstname != "Charlie" || lastname != "QueryRow" {
		t.Errorf("Expected Charlie QueryRow, got %s %s", firstname, lastname)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}
}

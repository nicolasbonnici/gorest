//go:build integration

package database_test

import (
	"context"
	"strings"
	"testing"

	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
	"github.com/nicolasbonnici/gorest/internal/testhelpers"
)

func TestErrors_InvalidDSN(t *testing.T) {
	tests := []struct {
		name   string
		driver string
		dsn    string
	}{
		{"postgres invalid", "postgres", "invalid://connection"},
		{"mysql invalid", "mysql", "invalid@connection"},
		{"sqlite invalid path", "sqlite", "/nonexistent/path/db.sqlite"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := database.Open(tt.driver, tt.dsn)
			if err == nil {
				db.Close()
				t.Error("Expected error for invalid DSN, got none")
			}
		})
	}
}

func TestErrors_ConnectionRefused(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
	}{
		{"postgres wrong port", "postgres://user:pass@localhost:9999/db"},
		{"mysql wrong port", "user:pass@tcp(localhost:9999)/db"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := database.Open("", tt.dsn)
			if err == nil {
				ctx := context.Background()
				if pingErr := db.Ping(ctx); pingErr == nil {
					db.Close()
					t.Error("Expected connection error, got none")
				}
				db.Close()
			}
		})
	}
}

func TestErrors_InvalidSQL(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(ctx, "INVALID SQL STATEMENT")
	if err == nil {
		t.Error("Expected syntax error, got none")
	}
}

func TestErrors_TableNotFound(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Query(ctx, "SELECT * FROM nonexistent_table")
	if err == nil {
		t.Error("Expected table not found error, got none")
	}
}

func TestErrors_UniqueConstraintViolation(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	defer db.Close()

	ctx := context.Background()
	testhelpers.CleanupDB(t, db)

	query := "INSERT INTO users (firstname, lastname, email) VALUES (" +
		db.Dialect().Placeholder(1) + ", " +
		db.Dialect().Placeholder(2) + ", " +
		db.Dialect().Placeholder(3) + ")"

	_, err := db.Exec(ctx, query, "Test", "User", "unique@example.com")
	if err != nil {
		t.Fatalf("Failed to insert first user: %v", err)
	}

	_, err = db.Exec(ctx, query, "Test", "User", "unique@example.com")
	if err == nil {
		t.Error("Expected unique constraint violation, got none")
	}
}

func TestErrors_ForeignKeyViolation(t *testing.T) {
	db := testhelpers.SetupSQLite(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(ctx, "PRAGMA foreign_keys = ON")
	if err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	_, err = db.Exec(ctx, "INSERT INTO todo (user_id, title, content) VALUES (?, ?, ?)",
		"nonexistent-user-id", "Test", "Content")
	if err == nil {
		t.Error("Expected foreign key violation, got none")
	}
}

func TestErrors_NullConstraintViolation(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	defer db.Close()

	ctx := context.Background()
	testhelpers.CleanupDB(t, db)

	_, err := db.Exec(ctx, "INSERT INTO users (email) VALUES ($1)", "null-test@example.com")
	if err == nil {
		t.Error("Expected NOT NULL constraint violation, got none")
	}
}

func TestErrors_TypeMismatch(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	defer db.Close()

	ctx := context.Background()
	testhelpers.CleanupDB(t, db)

	query := "INSERT INTO users (firstname, lastname, email) VALUES ($1, $2, $3)"
	_, err := db.Exec(ctx, query, "Test", "User", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	var wrongType int
	row := db.QueryRow(ctx, "SELECT firstname FROM users WHERE email = $1", "test@example.com")
	err = row.Scan(&wrongType)
	if err == nil {
		t.Error("Expected type mismatch error, got none")
	}
}

func TestErrors_TooManyColumns(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	defer db.Close()

	ctx := context.Background()
	testhelpers.CleanupDB(t, db)

	query := "INSERT INTO users (firstname, lastname, email) VALUES ($1, $2, $3)"
	_, err := db.Exec(ctx, query, "Test", "User", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	var firstname, lastname string
	row := db.QueryRow(ctx, "SELECT firstname, lastname, email FROM users WHERE email = $1", "test@example.com")
	err = row.Scan(&firstname, &lastname)
	if err == nil {
		t.Error("Expected too many columns error, got none")
	}
}

func TestErrors_ClosedConnection(t *testing.T) {
	db := testhelpers.SetupPostgres(t)

	db.Close()

	ctx := context.Background()
	_, err := db.Query(ctx, "SELECT 1")
	if err == nil {
		t.Error("Expected error on closed connection, got none")
	}
}

func TestErrors_ContextCanceled(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := db.Query(ctx, "SELECT pg_sleep(10)")
	if err == nil {
		t.Error("Expected context canceled error, got none")
	}

	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "cancel") {
		t.Logf("Got error: %v", err)
	}
}

func TestErrors_InvalidPlaceholderCount(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	defer db.Close()

	ctx := context.Background()

	_, err := db.Exec(ctx, "INSERT INTO users (firstname, lastname, email) VALUES ($1, $2, $3)", "Test", "User")
	if err == nil {
		t.Error("Expected wrong number of arguments error, got none")
	}
}

func TestErrors_QueryRowNoRows(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	defer db.Close()

	ctx := context.Background()
	testhelpers.CleanupDB(t, db)

	var email string
	row := db.QueryRow(ctx, "SELECT email FROM users WHERE email = $1", "nonexistent@example.com")
	err := row.Scan(&email)
	if err == nil {
		t.Error("Expected no rows error, got none")
	}
}

func TestErrors_TransactionAfterCommit(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	defer db.Close()

	ctx := context.Background()

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	_, err = tx.Exec(ctx, "SELECT 1")
	if err == nil {
		t.Error("Expected error using transaction after commit, got none")
	}
}

func TestErrors_DoubleCommit(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	defer db.Close()

	ctx := context.Background()

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	err = tx.Commit(ctx)
	if err == nil {
		t.Error("Expected error on double commit, got none")
	}
}

func TestErrors_RollbackAfterCommit(t *testing.T) {
	db := testhelpers.SetupPostgres(t)
	defer db.Close()

	ctx := context.Background()

	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	err = tx.Rollback(ctx)
	if err == nil {
		t.Error("Expected error on rollback after commit, got none")
	}
}

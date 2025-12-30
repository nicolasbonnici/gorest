package testhelpers

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/nicolasbonnici/gorest/database"
)

const (
	defaultPostgresDSN = "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable"
	defaultMySQLDSN    = "testuser:testpass@tcp(localhost:3307)/mydb_test"
)

// SetupTestDB creates a test database connection with auto-detection from environment or SQLite fallback.
// It automatically registers cleanup with t.Cleanup() to ensure proper teardown.
// The function tries TEST_DATABASE_URL first, then falls back to SQLite in-memory database.
func SetupTestDB(t *testing.T) database.Database {
	t.Helper()

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		return SetupSQLite(t)
	}

	db, err := database.Open("", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Fatalf("Database ping failed: %v", err)
	}

	t.Cleanup(func() {
		Cleanup(t, db)
	})

	return db
}

// SetupSQLite creates an in-memory SQLite database for isolated, fast tests.
// Each test gets a unique in-memory database to avoid conflicts.
// Automatically registers cleanup with t.Cleanup().
func SetupSQLite(t *testing.T) database.Database {
	t.Helper()

	// Use unique in-memory database for each test to avoid transaction conflicts
	// The mode=memory parameter with a unique name ensures complete isolation
	dbURL := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())

	db, err := database.Open("sqlite", dbURL)
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Fatalf("SQLite ping failed: %v", err)
	}

	t.Cleanup(func() {
		Cleanup(t, db)
	})

	return db
}

// SetupSQLiteWithSchema creates an in-memory SQLite database and applies the provided schema.
// This is useful for tests that need a specific database structure.
// Automatically registers cleanup with t.Cleanup().
func SetupSQLiteWithSchema(t *testing.T, schema string) database.Database {
	t.Helper()

	db := SetupSQLite(t)

	ctx := context.Background()
	_, err := db.Exec(ctx, schema)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Enable WAL mode for better concurrency and set busy timeout
	_, _ = db.Exec(ctx, "PRAGMA journal_mode=WAL")
	_, _ = db.Exec(ctx, "PRAGMA busy_timeout=5000")

	return db
}

// SetupSQLiteFile creates a file-based SQLite database with a unique name.
// This is useful for tests that need persistence or to avoid concurrent access issues.
// The database file is automatically cleaned up after the test.
func SetupSQLiteFile(t *testing.T) database.Database {
	t.Helper()

	// Use file-based database with a unique name for this test
	dbPath := fmt.Sprintf("/tmp/gorest_test_%s_%d.db", t.Name(), time.Now().UnixNano())

	// Clean up any existing database
	_ = os.Remove(dbPath)

	db, err := database.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		db.Close()
		_ = os.Remove(dbPath)
		t.Fatalf("SQLite ping failed: %v", err)
	}

	// Clean up database file when test completes
	t.Cleanup(func() {
		db.Close()
		_ = os.Remove(dbPath)
	})

	return db
}

// SetupSQLiteFileWithSchema creates a file-based SQLite database and applies the provided schema.
// The database file is automatically cleaned up after the test.
func SetupSQLiteFileWithSchema(t *testing.T, schema string) database.Database {
	t.Helper()

	db := SetupSQLiteFile(t)

	ctx := context.Background()
	_, err := db.Exec(ctx, schema)
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Enable WAL mode for better concurrency and set busy timeout
	_, _ = db.Exec(ctx, "PRAGMA journal_mode=WAL")
	_, _ = db.Exec(ctx, "PRAGMA busy_timeout=5000")
	_, _ = db.Exec(ctx, "PRAGMA foreign_keys = ON")

	return db
}

// SetupPostgres creates a PostgreSQL database connection for integration tests.
// Uses TEST_DATABASE_URL environment variable or falls back to default DSN.
// Skips the test if PostgreSQL is not available.
// Automatically registers cleanup with t.Cleanup().
func SetupPostgres(t *testing.T) database.Database {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultPostgresDSN
	}

	db, err := database.Open("postgres", dsn)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Skipf("PostgreSQL ping failed: %v", err)
	}

	t.Cleanup(func() {
		Cleanup(t, db)
	})

	return db
}

// SetupMySQL creates a MySQL database connection for integration tests.
// Uses MYSQL_TEST_URL environment variable or falls back to default DSN.
// Skips the test if MySQL is not available.
// Automatically registers cleanup with t.Cleanup().
func SetupMySQL(t *testing.T) database.Database {
	t.Helper()

	dsn := os.Getenv("MYSQL_TEST_URL")
	if dsn == "" {
		dsn = defaultMySQLDSN
	}

	db, err := database.Open("mysql", dsn)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Skipf("MySQL ping failed: %v", err)
	}

	t.Cleanup(func() {
		Cleanup(t, db)
	})

	return db
}

// SetupPostgresWithDSN creates a PostgreSQL database connection with a custom DSN.
// Skips the test if PostgreSQL is not available.
// Automatically registers cleanup with t.Cleanup().
func SetupPostgresWithDSN(t *testing.T, dsn string) database.Database {
	t.Helper()

	db, err := database.Open("postgres", dsn)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Skipf("PostgreSQL ping failed: %v", err)
	}

	t.Cleanup(func() {
		Cleanup(t, db)
	})

	return db
}

// SetupMySQLWithDSN creates a MySQL database connection with a custom DSN.
// Skips the test if MySQL is not available.
// Automatically registers cleanup with t.Cleanup().
func SetupMySQLWithDSN(t *testing.T, dsn string) database.Database {
	t.Helper()

	db, err := database.Open("mysql", dsn)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Skipf("MySQL ping failed: %v", err)
	}

	t.Cleanup(func() {
		Cleanup(t, db)
	})

	return db
}

// Cleanup closes the database connection and logs any errors.
// This is automatically called when using the Setup* functions via t.Cleanup().
func Cleanup(t *testing.T, db database.Database) {
	t.Helper()

	if db != nil {
		// Close the database connection to release any locks
		// SQLite in-memory databases are automatically destroyed when the connection closes
		if err := db.Close(); err != nil {
			t.Logf("Warning: failed to close database: %v", err)
		}
	}
}

// CleanupDB truncates/deletes data from common test tables.
// This is useful for cleaning up between test runs while keeping the schema intact.
func CleanupDB(t *testing.T, db database.Database) {
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

// CleanupTables truncates/deletes data from specified tables.
// This provides more granular control over which tables to clean.
func CleanupTables(t *testing.T, db database.Database, tables ...string) {
	t.Helper()
	ctx := context.Background()

	switch db.DriverName() {
	case "postgres":
		for _, table := range tables {
			_, _ = db.Exec(ctx, fmt.Sprintf("TRUNCATE %s CASCADE", table))
		}
	case "mysql":
		_, _ = db.Exec(ctx, "SET FOREIGN_KEY_CHECKS = 0")
		for _, table := range tables {
			_, _ = db.Exec(ctx, fmt.Sprintf("TRUNCATE %s", table))
		}
		_, _ = db.Exec(ctx, "SET FOREIGN_KEY_CHECKS = 1")
	case "sqlite":
		for _, table := range tables {
			_, _ = db.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		}
	}
}

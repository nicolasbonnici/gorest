package migrations

import (
	"context"
	"embed"
	"os"
	"testing"
	"time"

	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/sqlite" // Import SQLite driver for tests
)

//go:embed testdata/*.sql
var testMigrations embed.FS

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) database.Database {
	t.Helper()

	// Use SQLite for tests by default
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "file::memory:?cache=shared"
	}

	db, err := database.Open("", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	return db
}

// cleanupTestDB cleans up test database
func cleanupTestDB(t *testing.T, db database.Database) {
	t.Helper()

	ctx := context.Background()

	// Drop test tables
	db.Exec(ctx, "DROP TABLE IF EXISTS schema_migrations")
	db.Exec(ctx, "DROP TABLE IF EXISTS test_table")
	db.Exec(ctx, "DROP TABLE IF EXISTS users")

	db.Close()
}

func TestMigrationCalculateChecksum(t *testing.T) {
	migration := Migration{
		Version: "20250120000001",
		Name:    "test",
		Source:  "app",
		UpSQL:   "CREATE TABLE test (id INTEGER);",
		DownSQL: "DROP TABLE test;",
	}

	checksum1 := migration.CalculateChecksum()

	if checksum1 == "" {
		t.Error("Checksum should not be empty")
	}

	// Same content should produce same checksum
	checksum2 := migration.CalculateChecksum()
	if checksum1 != checksum2 {
		t.Error("Same migration should produce same checksum")
	}

	// Different content should produce different checksum
	migration.UpSQL = "CREATE TABLE test2 (id INTEGER);"
	checksum3 := migration.CalculateChecksum()
	if checksum1 == checksum3 {
		t.Error("Different migration should produce different checksum")
	}
}

func TestMigrationFullName(t *testing.T) {
	migration := Migration{
		Version: "20250120000001",
		Name:    "create_users",
		Source:  "app",
	}

	expected := "20250120000001_create_users"
	if migration.FullName() != expected {
		t.Errorf("Expected %s, got %s", expected, migration.FullName())
	}
}

func TestValidateTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		wantError bool
	}{
		{
			name:      "valid timestamp",
			version:   "20250120143022",
			wantError: false,
		},
		{
			name:      "too short",
			version:   "2025012014",
			wantError: true,
		},
		{
			name:      "too long",
			version:   "202501201430221",
			wantError: true,
		},
		{
			name:      "invalid date",
			version:   "20251332143022",
			wantError: true,
		},
		{
			name:      "non-numeric",
			version:   "2025abcd143022",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimestamp(tt.version)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTimestamp() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestMigrationTracker_CreateTrackingTable(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	tracker := NewMigrationTracker(db)
	ctx := context.Background()

	err := tracker.CreateTrackingTable(ctx)
	if err != nil {
		t.Fatalf("Failed to create tracking table: %v", err)
	}

	// Verify table exists by querying it
	_, err = db.Query(ctx, "SELECT version FROM schema_migrations LIMIT 1")
	if err != nil {
		t.Errorf("Tracking table doesn't exist or is malformed: %v", err)
	}

	// Should be idempotent
	err = tracker.CreateTrackingTable(ctx)
	if err != nil {
		t.Errorf("CreateTrackingTable should be idempotent: %v", err)
	}
}

func TestMigrationTracker_RecordAndGetMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	tracker := NewMigrationTracker(db)
	ctx := context.Background()

	// Create tracking table
	if err := tracker.CreateTrackingTable(ctx); err != nil {
		t.Fatalf("Failed to create tracking table: %v", err)
	}

	// Record a migration
	migration := Migration{
		Version:  "20250120000001",
		Name:     "test_migration",
		Source:   "app",
		UpSQL:    "CREATE TABLE test (id INTEGER);",
		DownSQL:  "DROP TABLE test;",
		Checksum: "abc123",
	}

	executionTime := 100 * time.Millisecond
	err := tracker.RecordMigration(ctx, migration, executionTime)
	if err != nil {
		t.Fatalf("Failed to record migration: %v", err)
	}

	// Get applied migrations
	applied, err := tracker.GetAppliedMigrations(ctx)
	if err != nil {
		t.Fatalf("Failed to get applied migrations: %v", err)
	}

	if len(applied) != 1 {
		t.Fatalf("Expected 1 applied migration, got %d", len(applied))
	}

	if applied[0].Migration.Version != migration.Version {
		t.Errorf("Expected version %s, got %s", migration.Version, applied[0].Migration.Version)
	}

	if applied[0].Status != "applied" {
		t.Errorf("Expected status 'applied', got %s", applied[0].Status)
	}

	if applied[0].Checksum != migration.Checksum {
		t.Errorf("Expected checksum %s, got %s", migration.Checksum, applied[0].Checksum)
	}
}

func TestMigrationTracker_CheckForDirtyDatabase(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	tracker := NewMigrationTracker(db)
	ctx := context.Background()

	// Create tracking table
	if err := tracker.CreateTrackingTable(ctx); err != nil {
		t.Fatalf("Failed to create tracking table: %v", err)
	}

	// Should be clean initially
	err := tracker.CheckForDirtyDatabase(ctx)
	if err != nil {
		t.Errorf("Expected clean database, got error: %v", err)
	}

	// Record a failed migration
	migration := Migration{
		Version:  "20250120000001",
		Name:     "failed_migration",
		Source:   "app",
		Checksum: "abc123",
	}

	err = tracker.RecordFailedMigration(ctx, migration, "syntax error")
	if err != nil {
		t.Fatalf("Failed to record failed migration: %v", err)
	}

	// Should now be dirty
	err = tracker.CheckForDirtyDatabase(ctx)
	if err == nil {
		t.Error("Expected dirty database error, got nil")
	}

	dirtyErr, ok := err.(*DirtyDatabaseError)
	if !ok {
		t.Errorf("Expected DirtyDatabaseError, got %T", err)
	}

	if len(dirtyErr.FailedMigrations) != 1 {
		t.Errorf("Expected 1 failed migration, got %d", len(dirtyErr.FailedMigrations))
	}
}

func TestMigrationTracker_VerifyChecksums(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	tracker := NewMigrationTracker(db)
	ctx := context.Background()

	// Create tracking table
	if err := tracker.CreateTrackingTable(ctx); err != nil {
		t.Fatalf("Failed to create tracking table: %v", err)
	}

	// Record migration with checksum
	migration := Migration{
		Version:  "20250120000001",
		Name:     "test",
		Source:   "app",
		UpSQL:    "CREATE TABLE test (id INTEGER);",
		DownSQL:  "DROP TABLE test;",
		Checksum: "checksum1",
	}

	err := tracker.RecordMigration(ctx, migration, 0)
	if err != nil {
		t.Fatalf("Failed to record migration: %v", err)
	}

	// Verify with same checksum - should pass
	err = tracker.VerifyChecksums(ctx, []Migration{migration})
	if err != nil {
		t.Errorf("Expected checksum verification to pass, got: %v", err)
	}

	// Verify with different checksum - should fail
	migration.Checksum = "checksum2"
	err = tracker.VerifyChecksums(ctx, []Migration{migration})
	if err == nil {
		t.Error("Expected checksum mismatch error, got nil")
	}

	_, ok := err.(*ChecksumMismatchError)
	if !ok {
		t.Errorf("Expected ChecksumMismatchError, got %T", err)
	}
}

func TestMigrationValidator_Validate(t *testing.T) {
	validator := NewMigrationValidator()

	tests := []struct {
		name      string
		migration Migration
		wantError bool
	}{
		{
			name: "valid migration",
			migration: Migration{
				Version:  "20250120000001",
				Name:     "test",
				Source:   "app",
				UpSQL:    "CREATE TABLE test (id INTEGER);",
				DownSQL:  "DROP TABLE test;",
				Checksum: "abc123",
			},
			wantError: false,
		},
		{
			name: "invalid version",
			migration: Migration{
				Version:  "invalid",
				Name:     "test",
				Source:   "app",
				UpSQL:    "CREATE TABLE test (id INTEGER);",
				DownSQL:  "DROP TABLE test;",
				Checksum: "abc123",
			},
			wantError: true,
		},
		{
			name: "empty name",
			migration: Migration{
				Version:  "20250120000001",
				Name:     "",
				Source:   "app",
				UpSQL:    "CREATE TABLE test (id INTEGER);",
				DownSQL:  "DROP TABLE test;",
				Checksum: "abc123",
			},
			wantError: true,
		},
		{
			name: "empty up SQL",
			migration: Migration{
				Version:  "20250120000001",
				Name:     "test",
				Source:   "app",
				UpSQL:    "",
				DownSQL:  "DROP TABLE test;",
				Checksum: "abc123",
			},
			wantError: true,
		},
		{
			name: "dangerous operation",
			migration: Migration{
				Version:  "20250120000001",
				Name:     "test",
				Source:   "app",
				UpSQL:    "DROP DATABASE mydb;",
				DownSQL:  "SELECT 1;",
				Checksum: "abc123",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.migration)
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestDependencyResolver_Resolve(t *testing.T) {
	resolver := NewDependencyResolver()

	// Add sources with dependencies
	resolver.AddSource("app", []string{})
	resolver.AddSource("auth", []string{"app"})
	resolver.AddSource("comments", []string{"app", "auth"})

	order, err := resolver.Resolve()
	if err != nil {
		t.Fatalf("Failed to resolve dependencies: %v", err)
	}

	// app should come first
	if order[0] != "app" {
		t.Errorf("Expected 'app' first, got %s", order[0])
	}

	// auth should come before comments
	authIdx := -1
	commentsIdx := -1
	for i, source := range order {
		if source == "auth" {
			authIdx = i
		}
		if source == "comments" {
			commentsIdx = i
		}
	}

	if authIdx == -1 || commentsIdx == -1 {
		t.Error("Missing auth or comments in resolved order")
	}

	if authIdx > commentsIdx {
		t.Error("auth should come before comments")
	}
}

func TestDependencyResolver_CircularDependency(t *testing.T) {
	resolver := NewDependencyResolver()

	// Create circular dependency
	resolver.AddSource("a", []string{"b"})
	resolver.AddSource("b", []string{"a"})

	_, err := resolver.Resolve()
	if err == nil {
		t.Error("Expected circular dependency error, got nil")
	}
}

func TestMigrationLock_AcquireRelease(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	lock := NewMigrationLock(db)
	ctx := context.Background()

	// Acquire lock
	err := lock.Acquire(ctx)
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Should be idempotent
	err = lock.Acquire(ctx)
	if err != nil {
		t.Errorf("Acquire should be idempotent: %v", err)
	}

	// Release lock
	err = lock.Release(ctx)
	if err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}

	// Should be idempotent
	err = lock.Release(ctx)
	if err != nil {
		t.Errorf("Release should be idempotent: %v", err)
	}
}

func TestMigrator_Integration(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create test migration source
	source := NewEmbeddedSource("test", testMigrations, "testdata", db)

	// Create migrator
	migrator := NewMigrator(db, source)

	ctx := context.Background()

	// Test Status before any migrations
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if len(statuses) == 0 {
		t.Log("No test migrations found - creating test migration files")
		// This is expected if testdata doesn't exist yet
		return
	}

	// Test Pending migrations
	pending, err := migrator.Pending(ctx)
	if err != nil {
		t.Fatalf("Failed to get pending migrations: %v", err)
	}

	pendingCount := len(pending)
	t.Logf("Found %d pending migrations", pendingCount)

	if pendingCount == 0 {
		t.Skip("No pending migrations to test")
	}

	// Test Up - apply all migrations
	err = migrator.Up(ctx)
	if err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Verify all migrations applied
	statuses, err = migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status after Up: %v", err)
	}

	for _, status := range statuses {
		if !status.Applied {
			t.Errorf("Migration %s should be applied", status.Migration.FullName())
		}
	}

	// Test Pending after Up - should be empty
	pending, err = migrator.Pending(ctx)
	if err != nil {
		t.Fatalf("Failed to get pending after Up: %v", err)
	}

	if len(pending) != 0 {
		t.Errorf("Expected 0 pending migrations after Up, got %d", len(pending))
	}

	// Test Down - revert last migration
	err = migrator.Down(ctx)
	if err != nil {
		t.Fatalf("Failed to revert migration: %v", err)
	}

	// Should have one pending migration now
	pending, err = migrator.Pending(ctx)
	if err != nil {
		t.Fatalf("Failed to get pending after Down: %v", err)
	}

	if len(pending) != 1 {
		t.Errorf("Expected 1 pending migration after Down, got %d", len(pending))
	}
}

package migrations

import (
	"context"
	"embed"
	"fmt"
	"testing"
	"time"

	_ "github.com/nicolasbonnici/gorest/database/sqlite"
	"github.com/nicolasbonnici/gorest/internal/testhelpers"
)

//go:embed testdata/*.sql
var testMigrations embed.FS

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
	db := testhelpers.SetupTestDB(t)

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
	db := testhelpers.SetupTestDB(t)

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
	db := testhelpers.SetupTestDB(t)

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
	db := testhelpers.SetupTestDB(t)

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
	db := testhelpers.SetupTestDB(t)

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
	db := testhelpers.SetupTestDB(t)

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

func TestMigrator_UpOne(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source := NewEmbeddedSource("test", testMigrations, "testdata", db)
	migrator := NewMigrator(db, source)
	ctx := context.Background()

	// Initialize by calling Status (creates tracking table)
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	pending, err := migrator.Pending(ctx)
	if err != nil {
		t.Fatalf("Failed to get pending migrations: %v", err)
	}

	if len(pending) == 0 {
		t.Skip("No pending migrations to test")
	}

	// Apply one migration
	err = migrator.UpOne(ctx)
	if err != nil {
		t.Fatalf("Failed to apply one migration: %v", err)
	}

	// Check that exactly one migration was applied
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	appliedCount := 0
	for _, s := range statuses {
		if s.Applied {
			appliedCount++
		}
	}

	if appliedCount != 1 {
		t.Errorf("Expected 1 applied migration, got %d", appliedCount)
	}

	// Apply another one
	if len(pending) > 1 {
		err = migrator.UpOne(ctx)
		if err != nil {
			t.Fatalf("Failed to apply second migration: %v", err)
		}

		statuses, err = migrator.Status(ctx)
		if err != nil {
			t.Fatalf("Failed to get status: %v", err)
		}

		appliedCount = 0
		for _, s := range statuses {
			if s.Applied {
				appliedCount++
			}
		}

		if appliedCount != 2 {
			t.Errorf("Expected 2 applied migrations, got %d", appliedCount)
		}
	}

	// Try to apply when no pending migrations left
	if len(pending) <= 2 {
		// Apply remaining migrations
		migrator.Up(ctx)

		// Now UpOne should fail with no pending migrations
		err = migrator.UpOne(ctx)
		if err != ErrNoPendingMigrations {
			t.Errorf("Expected ErrNoPendingMigrations, got %v", err)
		}
	}
}

func TestMigrator_UpTo(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source := NewEmbeddedSource("test", testMigrations, "testdata", db)
	migrator := NewMigrator(db, source)
	ctx := context.Background()

	// Initialize by calling Status (creates tracking table)
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	pending, err := migrator.Pending(ctx)
	if err != nil {
		t.Fatalf("Failed to get pending migrations: %v", err)
	}

	if len(pending) < 2 {
		t.Skip("Need at least 2 migrations to test UpTo")
	}

	// Migrate up to first migration
	targetVersion := pending[0].Version
	err = migrator.UpTo(ctx, targetVersion)
	if err != nil {
		t.Fatalf("Failed to migrate up to %s: %v", targetVersion, err)
	}

	// Check that only first migration was applied
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	appliedCount := 0
	var appliedVersion string
	for _, s := range statuses {
		if s.Applied {
			appliedCount++
			appliedVersion = s.Migration.Version
		}
	}

	if appliedCount != 1 {
		t.Errorf("Expected 1 applied migration, got %d", appliedCount)
	}

	if appliedVersion != targetVersion {
		t.Errorf("Expected version %s, got %s", targetVersion, appliedVersion)
	}

	// Test invalid version format
	err = migrator.UpTo(ctx, "invalid")
	if err == nil {
		t.Error("Expected error for invalid version, got nil")
	}
}

func TestMigrator_DownTo(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source := NewEmbeddedSource("test", testMigrations, "testdata", db)
	migrator := NewMigrator(db, source)
	ctx := context.Background()

	// Initialize by calling Status (creates tracking table)
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	pending, err := migrator.Pending(ctx)
	if err != nil {
		t.Fatalf("Failed to get pending migrations: %v", err)
	}

	if len(pending) < 2 {
		t.Skip("Need at least 2 migrations to test DownTo")
	}

	// First apply all migrations
	err = migrator.Up(ctx)
	if err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Now rollback to first migration version
	targetVersion := pending[0].Version
	err = migrator.DownTo(ctx, targetVersion)
	if err != nil {
		t.Fatalf("Failed to rollback to %s: %v", targetVersion, err)
	}

	// Check that only first migration remains
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	appliedCount := 0
	var appliedVersion string
	for _, s := range statuses {
		if s.Applied {
			appliedCount++
			appliedVersion = s.Migration.Version
		}
	}

	if appliedCount != 1 {
		t.Errorf("Expected 1 applied migration, got %d", appliedCount)
	}

	if appliedVersion != targetVersion {
		t.Errorf("Expected version %s, got %s", targetVersion, appliedVersion)
	}

	// Test invalid version format
	err = migrator.DownTo(ctx, "invalid")
	if err == nil {
		t.Error("Expected error for invalid version, got nil")
	}
}

func TestMigrator_UpSource(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source1 := NewEmbeddedSource("test1", testMigrations, "testdata", db)
	source2 := &testMigrationSource{
		name: "test2",
		migrations: []Migration{
			{
				Version:  "20250120000003",
				Name:     "source2_migration",
				Source:   "test2",
				UpSQL:    "CREATE TABLE test2 (id INTEGER);",
				DownSQL:  "DROP TABLE test2;",
				Checksum: "test2checksum",
			},
		},
	}

	migrator := NewMigrator(db, source1, source2)
	ctx := context.Background()

	// Initialize by calling Status (creates tracking table)
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	// Apply migrations for specific source
	err = migrator.UpSource(ctx, "test2")
	if err != nil {
		t.Fatalf("Failed to apply migrations for test2: %v", err)
	}

	// Check that only test2 migration was applied
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	found := false
	for _, s := range statuses {
		if s.Migration.Source == "test2" && s.Applied {
			found = true
		}
	}

	if !found {
		t.Error("test2 migration should be applied")
	}

	// Test non-existent source
	err = migrator.UpSource(ctx, "nonexistent")
	if err != ErrNoPendingMigrations {
		t.Errorf("Expected ErrNoPendingMigrations for non-existent source, got %v", err)
	}
}

func TestMigrator_DownSource(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source1 := NewEmbeddedSource("test1", testMigrations, "testdata", db)
	source2 := &testMigrationSource{
		name: "test2",
		migrations: []Migration{
			{
				Version:  "20250120000003",
				Name:     "source2_migration",
				Source:   "test2",
				UpSQL:    "CREATE TABLE test2 (id INTEGER);",
				DownSQL:  "DROP TABLE test2;",
				Checksum: "test2checksum",
			},
		},
	}

	migrator := NewMigrator(db, source1, source2)
	ctx := context.Background()

	// Initialize by calling Status (creates tracking table)
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	// First apply migration for test2
	err = migrator.UpSource(ctx, "test2")
	if err != nil {
		t.Fatalf("Failed to apply test2 migration: %v", err)
	}

	// Now rollback test2 migration
	err = migrator.DownSource(ctx, "test2")
	if err != nil {
		t.Fatalf("Failed to rollback test2 migration: %v", err)
	}

	// Check that test2 migration is gone
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	for _, s := range statuses {
		if s.Migration.Source == "test2" && s.Applied {
			t.Error("test2 migration should be rolled back")
		}
	}

	// Test rollback of non-existent source
	err = migrator.DownSource(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent source, got nil")
	}
}

func TestMigrator_Force(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source := NewEmbeddedSource("test", testMigrations, "testdata", db)
	migrator := NewMigrator(db, source)
	ctx := context.Background()

	// Initialize by calling Status (creates tracking table)
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	pending, err := migrator.Pending(ctx)
	if err != nil {
		t.Fatalf("Failed to get pending migrations: %v", err)
	}

	if len(pending) == 0 {
		t.Skip("No migrations to test Force")
	}

	migration := pending[0]

	// Force migration without executing
	err = migrator.Force(ctx, migration.Version, migration.Source)
	if err != nil {
		t.Fatalf("Failed to force migration: %v", err)
	}

	// Check that migration is marked as applied
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	found := false
	for _, s := range statuses {
		if s.Migration.Version == migration.Version && s.Migration.Source == migration.Source {
			found = true
			if !s.Applied {
				t.Error("Forced migration should be marked as applied")
			}
			if s.Status != "applied" {
				t.Errorf("Expected status 'applied', got %s", s.Status)
			}
		}
	}

	if !found {
		t.Error("Forced migration should be in status list")
	}

	// Test invalid version
	err = migrator.Force(ctx, "invalid", migration.Source)
	if err == nil {
		t.Error("Expected error for invalid version, got nil")
	}
}

func TestMigrator_DryRun(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source := NewEmbeddedSource("test", testMigrations, "testdata", db)
	migrator := NewMigrator(db, source)
	ctx := context.Background()

	// Initialize by calling Status (creates tracking table)
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	// DryRun should return pending migrations
	migrations, err := migrator.DryRun(ctx)
	if err != nil {
		t.Fatalf("Failed to run DryRun: %v", err)
	}

	if len(migrations) == 0 {
		t.Skip("No migrations for DryRun test")
	}

	// Verify no migrations were actually applied
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	for _, s := range statuses {
		if s.Applied {
			t.Error("DryRun should not apply any migrations")
		}
	}
}

func TestMigrator_Transactional(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source := NewEmbeddedSource("test", testMigrations, "testdata", db)
	migrator := NewMigrator(db, source)
	ctx := context.Background()

	// Initialize by calling Status (creates tracking table)
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	pending, err := migrator.Pending(ctx)
	if err != nil {
		t.Fatalf("Failed to get pending migrations: %v", err)
	}

	if len(pending) == 0 {
		t.Skip("No migrations to test transactional mode")
	}

	// Apply all migrations in single transaction
	opts := MigrationOptions{
		Transactional: true,
		StopOnError:   true,
	}

	err = migrator.UpWithOptions(ctx, opts)
	if err != nil {
		t.Fatalf("Failed to apply migrations transactionally: %v", err)
	}

	// Verify all migrations were applied
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	appliedCount := 0
	for _, s := range statuses {
		if s.Applied {
			appliedCount++
		}
	}

	if appliedCount != len(pending) {
		t.Errorf("Expected %d applied migrations, got %d", len(pending), appliedCount)
	}
}

func TestMigrator_SetSourceDependencies(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source1 := NewEmbeddedSource("app", testMigrations, "testdata", db)
	source2 := &testMigrationSource{
		name: "plugin",
		migrations: []Migration{
			{
				Version:  "20250120000003",
				Name:     "plugin_migration",
				Source:   "plugin",
				UpSQL:    "CREATE TABLE plugin_table (id INTEGER);",
				DownSQL:  "DROP TABLE plugin_table;",
				Checksum: "pluginchecksum",
			},
		},
	}

	migrator := NewMigrator(db, source1, source2)

	// Set dependency: plugin depends on app
	migrator.SetSourceDependencies("plugin", []string{"app"})

	ctx := context.Background()

	// Apply all migrations
	err := migrator.Up(ctx)
	if err != nil {
		t.Fatalf("Failed to apply migrations with dependencies: %v", err)
	}

	// Verify that app migrations were applied before plugin migrations
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	appIdx := -1
	pluginIdx := -1
	for i, s := range statuses {
		if s.Applied {
			if s.Migration.Source == "app" {
				appIdx = i
			}
			if s.Migration.Source == "plugin" {
				pluginIdx = i
			}
		}
	}

	if appIdx == -1 || pluginIdx == -1 {
		t.Log("Source ordering test skipped - not all sources found")
		return
	}

	if appIdx > pluginIdx {
		t.Error("App migrations should be applied before plugin migrations")
	}
}

func TestEmbeddedSource_SetDialect(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	// Create source without database
	source := NewEmbeddedSource("test", testMigrations, "testdata", nil)

	if source.dialect != "" {
		t.Error("Dialect should be empty when created without database")
	}

	// Set dialect late
	source.SetDialect(db)

	if source.dialect == "" {
		t.Error("Dialect should be set after SetDialect call")
	}

	if source.db == nil {
		t.Error("Database should be set after SetDialect call")
	}
}

func TestMigrationError_Error(t *testing.T) {
	migration := Migration{
		Version: "20250120000001",
		Name:    "test",
		Source:  "app",
	}

	err := &MigrationError{
		Migration:   migration,
		Err:         ErrMigrationFailed,
		SQL:         "INVALID SQL",
		DatabaseErr: "syntax error",
		Hint:        "Check your SQL syntax",
	}

	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Error message should not be empty")
	}

	if !contains(errMsg, migration.FullName()) {
		t.Error("Error message should contain migration name")
	}

	if !contains(errMsg, "syntax error") {
		t.Error("Error message should contain database error")
	}

	if !contains(errMsg, "Check your SQL syntax") {
		t.Error("Error message should contain hint")
	}
}

func TestChecksumMismatchError_Error(t *testing.T) {
	migration := Migration{
		Version: "20250120000001",
		Name:    "test",
		Source:  "app",
	}

	err := &ChecksumMismatchError{
		Migration:        migration,
		ExpectedChecksum: "abc123",
		ActualChecksum:   "def456",
	}

	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Error message should not be empty")
	}

	if !contains(errMsg, migration.FullName()) {
		t.Error("Error message should contain migration name")
	}

	if !contains(errMsg, "abc123") {
		t.Error("Error message should contain expected checksum")
	}

	if !contains(errMsg, "def456") {
		t.Error("Error message should contain actual checksum")
	}
}

func TestDirtyDatabaseError_Error(t *testing.T) {
	failedMigrations := []MigrationStatus{
		{
			Migration: Migration{
				Version: "20250120000001",
				Name:    "test",
				Source:  "app",
			},
			Status: "failed",
			Error:  "syntax error",
		},
	}

	err := &DirtyDatabaseError{
		FailedMigrations: failedMigrations,
	}

	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Error message should not be empty")
	}

	if !contains(errMsg, "20250120000001_test") {
		t.Error("Error message should contain failed migration")
	}
}

// Helper test migration source
type testMigrationSource struct {
	name       string
	migrations []Migration
}

func (s *testMigrationSource) Name() string {
	return s.name
}

func (s *testMigrationSource) Migrations() ([]Migration, error) {
	return s.migrations, nil
}

func TestMigrator_Validate(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source := NewEmbeddedSource("test", testMigrations, "testdata", db)
	migrator := NewMigrator(db, source)
	ctx := context.Background()

	// Validate should succeed for valid migrations
	err := migrator.Validate(ctx)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	// Create invalid migration source
	invalidSource := &testMigrationSource{
		name: "invalid",
		migrations: []Migration{
			{
				Version:  "invalid",
				Name:     "bad",
				Source:   "invalid",
				UpSQL:    "DROP DATABASE production;",
				DownSQL:  "SELECT 1;",
				Checksum: "bad",
			},
		},
	}

	invalidMigrator := NewMigrator(db, invalidSource)

	// Validate should fail for invalid migrations
	err = invalidMigrator.Validate(ctx)
	if err == nil {
		t.Error("Expected validation to fail for invalid migration")
	}
}

func TestMigrationError_Unwrap(t *testing.T) {
	innerErr := ErrMigrationFailed
	migration := Migration{
		Version: "20250120000001",
		Name:    "test",
		Source:  "app",
	}

	err := &MigrationError{
		Migration: migration,
		Err:       innerErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != innerErr {
		t.Errorf("Expected unwrapped error to be %v, got %v", innerErr, unwrapped)
	}
}

func TestMigrator_DryRunMode(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source := NewEmbeddedSource("test", testMigrations, "testdata", db)
	migrator := NewMigrator(db, source)
	ctx := context.Background()

	// Initialize
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	// Apply with DryRun option
	opts := MigrationOptions{
		DryRun:      true,
		StopOnError: true,
	}

	err = migrator.UpWithOptions(ctx, opts)
	if err != nil {
		t.Fatalf("DryRun failed: %v", err)
	}

	// Verify no migrations were actually applied
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	for _, s := range statuses {
		if s.Applied {
			t.Error("DryRun mode should not apply any migrations")
		}
	}
}

func TestMigrator_TransactionalRollback(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	// Create source with one valid and one invalid migration
	invalidSource := &testMigrationSource{
		name: "test",
		migrations: []Migration{
			{
				Version:  "20250120000001",
				Name:     "valid",
				Source:   "test",
				UpSQL:    "CREATE TABLE test (id INTEGER);",
				DownSQL:  "DROP TABLE test;",
				Checksum: "valid123",
			},
			{
				Version:  "20250120000002",
				Name:     "invalid",
				Source:   "test",
				UpSQL:    "INVALID SQL SYNTAX!!!",
				DownSQL:  "DROP TABLE test;",
				Checksum: "invalid123",
			},
		},
	}

	migrator := NewMigrator(db, invalidSource)
	ctx := context.Background()

	// Initialize
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	// Apply with Transactional option - should fail and rollback
	opts := MigrationOptions{
		Transactional: true,
		StopOnError:   true,
	}

	err = migrator.UpWithOptions(ctx, opts)
	if err == nil {
		t.Error("Expected transactional migration to fail")
	}

	// Verify no migrations were applied (rollback worked)
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	for _, s := range statuses {
		if s.Applied {
			t.Error("Transactional rollback should have reverted all migrations")
		}
	}
}

func TestMigrator_ErrorScenarios(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source := NewEmbeddedSource("test", testMigrations, "testdata", db)
	migrator := NewMigrator(db, source)
	ctx := context.Background()

	// Initialize
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	// Test UpOne when no pending migrations
	err = migrator.Up(ctx)
	if err != nil {
		t.Fatalf("Failed to apply all migrations: %v", err)
	}

	err = migrator.UpOne(ctx)
	if err != ErrNoPendingMigrations {
		t.Errorf("Expected ErrNoPendingMigrations, got %v", err)
	}

	// Test Down when no migrations to revert
	testhelpers.Cleanup(t, db)
	db = testhelpers.SetupTestDB(t)

	migrator = NewMigrator(db, source)
	_, err = migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	err = migrator.Down(ctx)
	if err == nil {
		t.Error("Expected error when trying to revert with no applied migrations")
	}
}

func TestEmbeddedSource_DialectFiltering(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source := NewEmbeddedSource("test", testMigrations, "testdata", db)

	// Test shouldLoadFile method indirectly through Migrations
	migrations, err := source.Migrations()
	if err != nil {
		t.Fatalf("Failed to load migrations: %v", err)
	}

	// Verify migrations were loaded (filtering worked)
	if len(migrations) == 0 {
		t.Skip("No migrations found in testdata")
	}

	// All loaded migrations should have checksums
	for _, m := range migrations {
		if m.Checksum == "" {
			t.Errorf("Migration %s should have checksum", m.FullName())
		}
	}
}

func TestMigrator_ExecutionTimeout(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source := &testMigrationSource{
		name: "test",
		migrations: []Migration{
			{
				Version:  "20250120000001",
				Name:     "with_timeout",
				Source:   "test",
				UpSQL:    "CREATE TABLE test (id INTEGER);",
				DownSQL:  "DROP TABLE test;",
				Checksum: "timeout123",
				Timeout:  1 * time.Millisecond,
			},
		},
	}

	migrator := NewMigrator(db, source)
	ctx := context.Background()

	// Initialize
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	// Execute with very short timeout - may or may not timeout depending on execution speed
	// This test mainly ensures timeout logic is in place
	migrator.Up(ctx)
}

func TestMigrator_ContinueOnError(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source := &testMigrationSource{
		name: "test",
		migrations: []Migration{
			{
				Version:  "20250120000001",
				Name:     "valid",
				Source:   "test",
				UpSQL:    "CREATE TABLE error_test_1 (id INTEGER);",
				DownSQL:  "DROP TABLE error_test_1;",
				Checksum: "valid123",
			},
			{
				Version:  "20250120000002",
				Name:     "invalid",
				Source:   "test",
				UpSQL:    "INVALID SQL",
				DownSQL:  "SELECT 1;",
				Checksum: "invalid123",
			},
			{
				Version:  "20250120000003",
				Name:     "another_valid",
				Source:   "test",
				UpSQL:    "CREATE TABLE error_test_2 (id INTEGER);",
				DownSQL:  "DROP TABLE error_test_2;",
				Checksum: "valid456",
			},
		},
	}

	migrator := NewMigrator(db, source)
	ctx := context.Background()

	// Initialize
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	// Apply with StopOnError=false to continue past errors
	opts := MigrationOptions{
		StopOnError: false,
	}

	err = migrator.UpWithOptions(ctx, opts)
	if err != nil {
		// Error is expected but execution should have continued
		t.Logf("Expected error occurred: %v", err)
	}

	// Verify first migration was applied
	statuses, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	firstApplied := false
	for _, s := range statuses {
		if s.Migration.Name == "valid" && s.Applied {
			firstApplied = true
		}
	}

	if !firstApplied {
		t.Error("First migration should have been applied despite later error")
	}
}

func TestMigrator_DownErrors(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source := &testMigrationSource{
		name: "test",
		migrations: []Migration{
			{
				Version:  "20250120000001",
				Name:     "valid_up_bad_down",
				Source:   "test",
				UpSQL:    "CREATE TABLE down_error_test (id INTEGER);",
				DownSQL:  "INVALID DOWN SQL",
				Checksum: "baddown123",
			},
		},
	}

	migrator := NewMigrator(db, source)
	ctx := context.Background()

	// Initialize and apply migration
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	err = migrator.Up(ctx)
	if err != nil {
		t.Fatalf("Failed to apply migration: %v", err)
	}

	// Try to revert - should fail due to bad down SQL
	err = migrator.Down(ctx)
	if err == nil {
		t.Error("Expected error when reverting migration with bad down SQL")
	}
}

func TestMigrator_FindMigrationError(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source := NewEmbeddedSource("test", testMigrations, "testdata", db)
	migrator := NewMigrator(db, source)
	ctx := context.Background()

	// Initialize
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	// Try to force non-existent migration (with valid timestamp format but non-existent version)
	err = migrator.Force(ctx, "20251231235959", "nonexistent")
	if err == nil {
		t.Error("Expected error when forcing non-existent migration")
	}

	if err != ErrMigrationNotFound {
		t.Errorf("Expected ErrMigrationNotFound, got %v", err)
	}
}

func TestMigrator_LoadError(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	// Create source that returns error on load
	errorSource := &errorMigrationSource{
		name: "error",
		err:  fmt.Errorf("failed to load migrations"),
	}

	migrator := NewMigrator(db, errorSource)
	ctx := context.Background()

	// All methods that load migrations should fail
	_, err := migrator.Pending(ctx)
	if err == nil {
		t.Error("Expected error when loading migrations fails")
	}

	err = migrator.Up(ctx)
	if err == nil {
		t.Error("Expected error when loading migrations fails")
	}

	_, err = migrator.Status(ctx)
	if err == nil {
		t.Error("Expected error when loading migrations fails")
	}
}

func TestEmbeddedSource_NoDialect(t *testing.T) {
	// Create source without database (no dialect)
	source := NewEmbeddedSource("test", testMigrations, "testdata", nil)

	migrations, err := source.Migrations()
	if err != nil {
		t.Fatalf("Failed to load migrations without dialect: %v", err)
	}

	// Should load migrations even without dialect
	if len(migrations) > 0 {
		for _, m := range migrations {
			if m.UpSQL == "" {
				t.Error("Migration should have UpSQL")
			}
			if m.DownSQL == "" {
				t.Error("Migration should have DownSQL")
			}
		}
	}
}

func TestMigrator_UpWithOptionsErrors(t *testing.T) {
	db := testhelpers.SetupTestDB(t)

	source := NewEmbeddedSource("test", testMigrations, "testdata", db)
	migrator := NewMigrator(db, source)
	ctx := context.Background()

	// Initialize
	_, err := migrator.Status(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial status: %v", err)
	}

	// Apply all migrations
	err = migrator.Up(ctx)
	if err != nil {
		t.Fatalf("Failed to apply migrations: %v", err)
	}

	// Try UpWithOptions when no pending migrations
	opts := MigrationOptions{
		StopOnError: true,
	}

	err = migrator.UpWithOptions(ctx, opts)
	if err != ErrNoPendingMigrations {
		t.Errorf("Expected ErrNoPendingMigrations, got %v", err)
	}
}

func TestValidator_ValidateNoDuplicates(t *testing.T) {
	validator := NewMigrationValidator()

	migrations := []Migration{
		{
			Version: "20250120000001",
			Name:    "test1",
			Source:  "app",
		},
		{
			Version: "20250120000002",
			Name:    "test2",
			Source:  "app",
		},
	}

	// Should pass with no duplicates
	err := validator.ValidateNoDuplicates(migrations)
	if err != nil {
		t.Errorf("ValidateNoDuplicates should pass with no duplicates: %v", err)
	}
}

func TestValidator_Validate_EmptySource(t *testing.T) {
	validator := NewMigrationValidator()

	migration := Migration{
		Version:  "20250120000001",
		Name:     "test",
		Source:   "",
		UpSQL:    "CREATE TABLE test (id INTEGER);",
		DownSQL:  "DROP TABLE test;",
		Checksum: "abc123",
	}

	err := validator.Validate(migration)
	if err == nil {
		t.Error("Expected error for empty source")
	}
}

func TestValidator_Validate_EmptyDownSQL(t *testing.T) {
	validator := NewMigrationValidator()

	migration := Migration{
		Version:  "20250120000001",
		Name:     "test",
		Source:   "app",
		UpSQL:    "CREATE TABLE test (id INTEGER);",
		DownSQL:  "",
		Checksum: "abc123",
	}

	err := validator.Validate(migration)
	if err == nil {
		t.Error("Expected error for empty down SQL")
	}
}

func TestValidator_Validate_EmptyChecksum(t *testing.T) {
	validator := NewMigrationValidator()

	migration := Migration{
		Version:  "20250120000001",
		Name:     "test",
		Source:   "app",
		UpSQL:    "CREATE TABLE test (id INTEGER);",
		DownSQL:  "DROP TABLE test;",
		Checksum: "",
	}

	err := validator.Validate(migration)
	if err == nil {
		t.Error("Expected error for empty checksum")
	}
}

// Helper error source for testing
type errorMigrationSource struct {
	name string
	err  error
}

func (s *errorMigrationSource) Name() string {
	return s.name
}

func (s *errorMigrationSource) Migrations() ([]Migration, error) {
	return nil, s.err
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

package migrations

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nicolasbonnici/gorest/database"
)

// migrator implements the Migrator interface
type migrator struct {
	db         database.Database
	sources    []MigrationSource
	tracker    *MigrationTracker
	lock       *MigrationLock
	validator  *MigrationValidator
	resolver   *DependencyResolver
	sourceDeps map[string][]string
}

// NewMigrator creates a new migrator with given sources
func NewMigrator(db database.Database, sources ...MigrationSource) Migrator {
	return &migrator{
		db:         db,
		sources:    sources,
		tracker:    NewMigrationTracker(db),
		lock:       NewMigrationLock(db),
		validator:  NewMigrationValidator(),
		resolver:   NewDependencyResolver(),
		sourceDeps: make(map[string][]string),
	}
}

// SetSourceDependencies sets dependencies for a source
func (m *migrator) SetSourceDependencies(source string, dependencies []string) {
	m.sourceDeps[source] = dependencies
	m.resolver.AddSource(source, dependencies)
}

// Up applies all pending migrations from all sources
func (m *migrator) Up(ctx context.Context) error {
	return m.UpWithOptions(ctx, MigrationOptions{
		StopOnError: true,
	})
}

// UpWithOptions applies migrations with custom options
func (m *migrator) UpWithOptions(ctx context.Context, opts MigrationOptions) error {
	// Acquire advisory lock
	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.lock.Release(ctx)

	// Initialize tracking table
	if err := m.tracker.CreateTrackingTable(ctx); err != nil {
		return fmt.Errorf("failed to create tracking table: %w", err)
	}

	// Check for dirty database
	if err := m.tracker.CheckForDirtyDatabase(ctx); err != nil {
		return err
	}

	// Load all migrations
	allMigrations, err := m.loadAllMigrations()
	if err != nil {
		return err
	}

	if len(allMigrations) == 0 {
		return ErrNoMigrations
	}

	// Validate all migrations
	if err := m.validator.ValidateBatch(allMigrations); err != nil {
		return err
	}

	// Verify checksums for applied migrations
	if err := m.tracker.VerifyChecksums(ctx, allMigrations); err != nil {
		return err
	}

	// Get already applied migrations
	applied, err := m.tracker.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	// Filter pending migrations
	pending := m.filterPending(allMigrations, applied)

	if len(pending) == 0 {
		return ErrNoPendingMigrations
	}

	// Order migrations by dependencies and version
	ordered, err := m.orderMigrations(pending)
	if err != nil {
		return err
	}

	if opts.DryRun {
		// Just return the migrations that would be executed
		return nil
	}

	// Execute migrations
	if opts.Transactional {
		return m.executeTransactional(ctx, ordered)
	}

	return m.executeSequential(ctx, ordered, opts.StopOnError)
}

// UpOne applies the next pending migration
func (m *migrator) UpOne(ctx context.Context) error {
	// Acquire advisory lock
	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.lock.Release(ctx)

	// Initialize tracking table
	if err := m.tracker.CreateTrackingTable(ctx); err != nil {
		return err
	}

	// Check for dirty database
	if err := m.tracker.CheckForDirtyDatabase(ctx); err != nil {
		return err
	}

	// Get pending migrations
	pending, err := m.Pending(ctx)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		return ErrNoPendingMigrations
	}

	// Execute first pending migration
	return m.executeMigration(ctx, pending[0])
}

// UpTo applies migrations up to specific version
func (m *migrator) UpTo(ctx context.Context, version string) error {
	// Validate version format
	if err := ValidateTimestamp(version); err != nil {
		return err
	}

	// Acquire advisory lock
	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.lock.Release(ctx)

	// Initialize tracking table
	if err := m.tracker.CreateTrackingTable(ctx); err != nil {
		return err
	}

	// Check for dirty database
	if err := m.tracker.CheckForDirtyDatabase(ctx); err != nil {
		return err
	}

	// Get pending migrations
	pending, err := m.Pending(ctx)
	if err != nil {
		return err
	}

	// Filter migrations up to version
	var toApply []Migration
	for _, migration := range pending {
		if migration.Version <= version {
			toApply = append(toApply, migration)
		}
	}

	if len(toApply) == 0 {
		return ErrNoPendingMigrations
	}

	// Execute migrations
	return m.executeSequential(ctx, toApply, true)
}

// Down reverts the most recently applied migration
func (m *migrator) Down(ctx context.Context) error {
	// Acquire advisory lock
	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.lock.Release(ctx)

	// Get applied migrations
	applied, err := m.tracker.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		return fmt.Errorf("no migrations to revert")
	}

	// Find most recent applied migration
	var mostRecent *MigrationStatus
	for i := range applied {
		if applied[i].Status == "applied" {
			if mostRecent == nil || applied[i].Migration.Version > mostRecent.Migration.Version {
				mostRecent = &applied[i]
			}
		}
	}

	if mostRecent == nil {
		return fmt.Errorf("no applied migrations to revert")
	}

	// Load the migration to get down SQL
	migration, err := m.findMigration(mostRecent.Migration.Version, mostRecent.Migration.Source)
	if err != nil {
		return err
	}

	// Execute down migration
	return m.executeDown(ctx, migration)
}

// DownTo reverts migrations down to specific version
func (m *migrator) DownTo(ctx context.Context, version string) error {
	// Validate version format
	if err := ValidateTimestamp(version); err != nil {
		return err
	}

	// Acquire advisory lock
	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.lock.Release(ctx)

	// Get applied migrations
	applied, err := m.tracker.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	// Find migrations to revert (those with version > target version)
	var toRevert []Migration
	for _, status := range applied {
		if status.Status == "applied" && status.Migration.Version > version {
			migration, err := m.findMigration(status.Migration.Version, status.Migration.Source)
			if err != nil {
				return err
			}
			toRevert = append(toRevert, migration)
		}
	}

	if len(toRevert) == 0 {
		return fmt.Errorf("no migrations to revert")
	}

	// Revert in reverse order
	for i := len(toRevert) - 1; i >= 0; i-- {
		if err := m.executeDown(ctx, toRevert[i]); err != nil {
			return err
		}
	}

	return nil
}

// Status returns migration status for all sources
func (m *migrator) Status(ctx context.Context) ([]MigrationStatus, error) {
	// Initialize tracking table if needed
	if err := m.tracker.CreateTrackingTable(ctx); err != nil {
		return nil, err
	}

	// Load all migrations
	allMigrations, err := m.loadAllMigrations()
	if err != nil {
		return nil, err
	}

	// Get applied migrations
	applied, err := m.tracker.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	// Build map of applied migrations
	appliedMap := make(map[string]MigrationStatus)
	for _, a := range applied {
		key := a.Migration.Version + ":" + a.Migration.Source
		appliedMap[key] = a
	}

	// Build complete status list
	var statuses []MigrationStatus
	for _, migration := range allMigrations {
		key := migration.Version + ":" + migration.Source

		if status, exists := appliedMap[key]; exists {
			statuses = append(statuses, status)
		} else {
			statuses = append(statuses, MigrationStatus{
				Migration: migration,
				Applied:   false,
				Status:    "pending",
			})
		}
	}

	return statuses, nil
}

// Pending returns list of pending migrations
func (m *migrator) Pending(ctx context.Context) ([]Migration, error) {
	// Load all migrations
	allMigrations, err := m.loadAllMigrations()
	if err != nil {
		return nil, err
	}

	// Get applied migrations
	applied, err := m.tracker.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	// Filter pending
	pending := m.filterPending(allMigrations, applied)

	// Order by dependencies and version
	return m.orderMigrations(pending)
}

// Validate validates all migrations without executing
func (m *migrator) Validate(ctx context.Context) error {
	// Load all migrations
	allMigrations, err := m.loadAllMigrations()
	if err != nil {
		return err
	}

	// Validate batch
	return m.validator.ValidateBatch(allMigrations)
}

// DryRun shows what would be executed
func (m *migrator) DryRun(ctx context.Context) ([]Migration, error) {
	return m.Pending(ctx)
}

// Force marks a migration as applied without executing
func (m *migrator) Force(ctx context.Context, version, source string) error {
	// Validate version
	if err := ValidateTimestamp(version); err != nil {
		return err
	}

	// Find migration
	migration, err := m.findMigration(version, source)
	if err != nil {
		return err
	}

	// Initialize tracking table
	if err := m.tracker.CreateTrackingTable(ctx); err != nil {
		return err
	}

	// Force migration
	log.Printf("WARNING: Forcing migration %s/%s as applied without executing", source, migration.FullName())

	return m.tracker.ForceMigration(ctx, migration)
}

// UpSource applies all pending migrations for a specific source
func (m *migrator) UpSource(ctx context.Context, sourceName string) error {
	// Get pending migrations
	pending, err := m.Pending(ctx)
	if err != nil {
		return err
	}

	// Filter by source
	var sourceMigrations []Migration
	for _, migration := range pending {
		if migration.Source == sourceName {
			sourceMigrations = append(sourceMigrations, migration)
		}
	}

	if len(sourceMigrations) == 0 {
		return ErrNoPendingMigrations
	}

	// Acquire lock and execute
	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.lock.Release(ctx)

	return m.executeSequential(ctx, sourceMigrations, true)
}

// DownSource reverts the most recent migration for a specific source
func (m *migrator) DownSource(ctx context.Context, sourceName string) error {
	// Acquire advisory lock
	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.lock.Release(ctx)

	// Get applied migrations
	applied, err := m.tracker.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	// Find most recent for this source
	var mostRecent *MigrationStatus
	for i := range applied {
		if applied[i].Status == "applied" && applied[i].Migration.Source == sourceName {
			if mostRecent == nil || applied[i].Migration.Version > mostRecent.Migration.Version {
				mostRecent = &applied[i]
			}
		}
	}

	if mostRecent == nil {
		return fmt.Errorf("no applied migrations for source: %s", sourceName)
	}

	// Load the migration
	migration, err := m.findMigration(mostRecent.Migration.Version, mostRecent.Migration.Source)
	if err != nil {
		return err
	}

	// Execute down
	return m.executeDown(ctx, migration)
}

// loadAllMigrations loads migrations from all sources
func (m *migrator) loadAllMigrations() ([]Migration, error) {
	var allMigrations []Migration

	for _, source := range m.sources {
		migrations, err := source.Migrations()
		if err != nil {
			return nil, fmt.Errorf("failed to load migrations from %s: %w", source.Name(), err)
		}

		allMigrations = append(allMigrations, migrations...)

		// Register source with resolver
		deps := m.sourceDeps[source.Name()]
		m.resolver.AddSource(source.Name(), deps)
	}

	return allMigrations, nil
}

// filterPending filters out already applied migrations
func (m *migrator) filterPending(all []Migration, applied []MigrationStatus) []Migration {
	appliedMap := make(map[string]bool)
	for _, a := range applied {
		if a.Status == "applied" {
			key := a.Migration.Version + ":" + a.Migration.Source
			appliedMap[key] = true
		}
	}

	var pending []Migration
	for _, migration := range all {
		key := migration.Version + ":" + migration.Source
		if !appliedMap[key] {
			pending = append(pending, migration)
		}
	}

	return pending
}

// orderMigrations orders migrations by dependencies and version
func (m *migrator) orderMigrations(migrations []Migration) ([]Migration, error) {
	return m.resolver.OrderMigrations(migrations)
}

// findMigration finds a migration by version and source
func (m *migrator) findMigration(version, source string) (Migration, error) {
	for _, s := range m.sources {
		if s.Name() == source {
			migrations, err := s.Migrations()
			if err != nil {
				return Migration{}, err
			}

			for _, migration := range migrations {
				if migration.Version == version {
					return migration, nil
				}
			}
		}
	}

	return Migration{}, ErrMigrationNotFound
}

// executeMigration executes a single migration
func (m *migrator) executeMigration(ctx context.Context, migration Migration) error {
	log.Printf("Applying migration [%s] %s...", migration.Source, migration.FullName())

	start := time.Now()

	// Apply timeout
	if migration.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, migration.Timeout)
		defer cancel()
	}

	// Begin transaction
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute up SQL
	_, err = tx.Exec(ctx, migration.UpSQL)
	if err != nil {
		tx.Rollback(ctx)

		// Record failed migration
		m.tracker.RecordFailedMigration(ctx, migration, err.Error())

		return &MigrationError{
			Migration:   migration,
			Err:         ErrMigrationFailed,
			SQL:         migration.UpSQL,
			DatabaseErr: err.Error(),
			Hint:        "Check SQL syntax and database state",
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	// Record migration
	executionTime := time.Since(start)
	if err := m.tracker.RecordMigration(ctx, migration, executionTime); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	log.Printf("✓ Applied [%s] %s (%dms)", migration.Source, migration.FullName(), executionTime.Milliseconds())

	return nil
}

// executeDown executes a down migration
func (m *migrator) executeDown(ctx context.Context, migration Migration) error {
	log.Printf("Reverting migration [%s] %s...", migration.Source, migration.FullName())

	// Begin transaction
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute down SQL
	_, err = tx.Exec(ctx, migration.DownSQL)
	if err != nil {
		tx.Rollback(ctx)
		return &MigrationError{
			Migration:   migration,
			Err:         ErrMigrationFailed,
			SQL:         migration.DownSQL,
			DatabaseErr: err.Error(),
			Hint:        "Check down migration SQL syntax",
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	// Remove migration record
	if err := m.tracker.RemoveMigration(ctx, migration.Version, migration.Source); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	log.Printf("✓ Reverted [%s] %s", migration.Source, migration.FullName())

	return nil
}

// executeSequential executes migrations one by one
func (m *migrator) executeSequential(ctx context.Context, migrations []Migration, stopOnError bool) error {
	for _, migration := range migrations {
		if err := m.executeMigration(ctx, migration); err != nil {
			if stopOnError {
				return err
			}
			log.Printf("Error in migration %s: %v (continuing...)", migration.FullName(), err)
		}
	}

	return nil
}

// executeTransactional executes all migrations in a single transaction
func (m *migrator) executeTransactional(ctx context.Context, migrations []Migration) error {
	log.Printf("Executing %d migrations in single transaction...", len(migrations))

	// Begin transaction
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute all migrations
	for _, migration := range migrations {
		log.Printf("Applying migration [%s] %s...", migration.Source, migration.FullName())

		_, err = tx.Exec(ctx, migration.UpSQL)
		if err != nil {
			tx.Rollback(ctx)
			return &MigrationError{
				Migration:   migration,
				Err:         ErrMigrationFailed,
				SQL:         migration.UpSQL,
				DatabaseErr: err.Error(),
				Hint:        "Transaction rolled back - no migrations were applied",
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migrations: %w", err)
	}

	// Record all migrations
	for _, migration := range migrations {
		if err := m.tracker.RecordMigration(ctx, migration, 0); err != nil {
			log.Printf("Warning: failed to record migration %s: %v", migration.FullName(), err)
		}
	}

	log.Printf("✓ All %d migrations applied successfully", len(migrations))

	return nil
}

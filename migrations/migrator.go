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

func (m *migrator) SetSourceDependencies(source string, dependencies []string) {
	m.sourceDeps[source] = dependencies
	m.resolver.AddSource(source, dependencies)
}

func (m *migrator) Up(ctx context.Context) error {
	return m.UpWithOptions(ctx, MigrationOptions{
		StopOnError: true,
	})
}

func (m *migrator) UpWithOptions(ctx context.Context, opts MigrationOptions) error {
	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.lock.Release(ctx)

	if err := m.tracker.CreateTrackingTable(ctx); err != nil {
		return fmt.Errorf("failed to create tracking table: %w", err)
	}

	if err := m.tracker.CheckForDirtyDatabase(ctx); err != nil {
		return err
	}

	allMigrations, err := m.loadAllMigrations()
	if err != nil {
		return err
	}

	if len(allMigrations) == 0 {
		return ErrNoMigrations
	}

	if err := m.validator.ValidateBatch(allMigrations); err != nil {
		return err
	}

	if err := m.tracker.VerifyChecksums(ctx, allMigrations); err != nil {
		return err
	}

	applied, err := m.tracker.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	pending := m.filterPending(allMigrations, applied)

	if len(pending) == 0 {
		return ErrNoPendingMigrations
	}

	ordered, err := m.orderMigrations(pending)
	if err != nil {
		return err
	}

	if opts.DryRun {
		return nil
	}

	if opts.Transactional {
		return m.executeTransactional(ctx, ordered)
	}

	return m.executeSequential(ctx, ordered, opts.StopOnError)
}

func (m *migrator) UpOne(ctx context.Context) error {
	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.lock.Release(ctx)

	if err := m.tracker.CreateTrackingTable(ctx); err != nil {
		return err
	}

	if err := m.tracker.CheckForDirtyDatabase(ctx); err != nil {
		return err
	}

	pending, err := m.Pending(ctx)
	if err != nil {
		return err
	}

	if len(pending) == 0 {
		return ErrNoPendingMigrations
	}

	return m.executeMigration(ctx, pending[0])
}

func (m *migrator) UpTo(ctx context.Context, version string) error {
	if err := ValidateTimestamp(version); err != nil {
		return err
	}

	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.lock.Release(ctx)

	if err := m.tracker.CreateTrackingTable(ctx); err != nil {
		return err
	}

	if err := m.tracker.CheckForDirtyDatabase(ctx); err != nil {
		return err
	}

	pending, err := m.Pending(ctx)
	if err != nil {
		return err
	}

	var toApply []Migration
	for _, migration := range pending {
		if migration.Version <= version {
			toApply = append(toApply, migration)
		}
	}

	if len(toApply) == 0 {
		return ErrNoPendingMigrations
	}

	return m.executeSequential(ctx, toApply, true)
}

func (m *migrator) Down(ctx context.Context) error {
	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.lock.Release(ctx)

	applied, err := m.tracker.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		return fmt.Errorf("no migrations to revert")
	}

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

	migration, err := m.findMigration(mostRecent.Migration.Version, mostRecent.Migration.Source)
	if err != nil {
		return err
	}

	return m.executeDown(ctx, migration)
}

func (m *migrator) DownTo(ctx context.Context, version string) error {
	if err := ValidateTimestamp(version); err != nil {
		return err
	}

	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.lock.Release(ctx)

	applied, err := m.tracker.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

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

	for i := len(toRevert) - 1; i >= 0; i-- {
		if err := m.executeDown(ctx, toRevert[i]); err != nil {
			return err
		}
	}

	return nil
}

func (m *migrator) Status(ctx context.Context) ([]MigrationStatus, error) {
	if err := m.tracker.CreateTrackingTable(ctx); err != nil {
		return nil, err
	}

	allMigrations, err := m.loadAllMigrations()
	if err != nil {
		return nil, err
	}

	applied, err := m.tracker.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	appliedMap := make(map[string]MigrationStatus)
	for _, a := range applied {
		key := a.Migration.Version + ":" + a.Migration.Source
		appliedMap[key] = a
	}

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

func (m *migrator) Pending(ctx context.Context) ([]Migration, error) {
	allMigrations, err := m.loadAllMigrations()
	if err != nil {
		return nil, err
	}

	applied, err := m.tracker.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	pending := m.filterPending(allMigrations, applied)

	return m.orderMigrations(pending)
}

func (m *migrator) Validate(ctx context.Context) error {
	allMigrations, err := m.loadAllMigrations()
	if err != nil {
		return err
	}

	return m.validator.ValidateBatch(allMigrations)
}

func (m *migrator) DryRun(ctx context.Context) ([]Migration, error) {
	return m.Pending(ctx)
}

func (m *migrator) Force(ctx context.Context, version, source string) error {
	if err := ValidateTimestamp(version); err != nil {
		return err
	}

	migration, err := m.findMigration(version, source)
	if err != nil {
		return err
	}

	if err := m.tracker.CreateTrackingTable(ctx); err != nil {
		return err
	}

	log.Printf("WARNING: Forcing migration %s/%s as applied without executing", source, migration.FullName())

	return m.tracker.ForceMigration(ctx, migration)
}

func (m *migrator) UpSource(ctx context.Context, sourceName string) error {
	pending, err := m.Pending(ctx)
	if err != nil {
		return err
	}

	var sourceMigrations []Migration
	for _, migration := range pending {
		if migration.Source == sourceName {
			sourceMigrations = append(sourceMigrations, migration)
		}
	}

	if len(sourceMigrations) == 0 {
		return ErrNoPendingMigrations
	}

	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.lock.Release(ctx)

	return m.executeSequential(ctx, sourceMigrations, true)
}

func (m *migrator) DownSource(ctx context.Context, sourceName string) error {
	if err := m.lock.Acquire(ctx); err != nil {
		return err
	}
	defer m.lock.Release(ctx)

	applied, err := m.tracker.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

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

	migration, err := m.findMigration(mostRecent.Migration.Version, mostRecent.Migration.Source)
	if err != nil {
		return err
	}

	return m.executeDown(ctx, migration)
}

func (m *migrator) loadAllMigrations() ([]Migration, error) {
	var allMigrations []Migration

	for _, source := range m.sources {
		migrations, err := source.Migrations()
		if err != nil {
			return nil, fmt.Errorf("failed to load migrations from %s: %w", source.Name(), err)
		}

		allMigrations = append(allMigrations, migrations...)

		deps := m.sourceDeps[source.Name()]
		m.resolver.AddSource(source.Name(), deps)
	}

	return allMigrations, nil
}

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

func (m *migrator) orderMigrations(migrations []Migration) ([]Migration, error) {
	return m.resolver.OrderMigrations(migrations)
}

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

func (m *migrator) executeMigration(ctx context.Context, migration Migration) error {
	log.Printf("Applying migration [%s] %s...", migration.Source, migration.FullName())

	start := time.Now()

	if migration.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, migration.Timeout)
		defer cancel()
	}

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	_, err = tx.Exec(ctx, migration.UpSQL)
	if err != nil {
		tx.Rollback(ctx)

		m.tracker.RecordFailedMigration(ctx, migration, err.Error())

		return &MigrationError{
			Migration:   migration,
			Err:         ErrMigrationFailed,
			SQL:         migration.UpSQL,
			DatabaseErr: err.Error(),
			Hint:        "Check SQL syntax and database state",
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	executionTime := time.Since(start)
	if err := m.tracker.RecordMigration(ctx, migration, executionTime); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	log.Printf("✓ Applied [%s] %s (%dms)", migration.Source, migration.FullName(), executionTime.Milliseconds())

	return nil
}

func (m *migrator) executeDown(ctx context.Context, migration Migration) error {
	log.Printf("Reverting migration [%s] %s...", migration.Source, migration.FullName())

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

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

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	if err := m.tracker.RemoveMigration(ctx, migration.Version, migration.Source); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	log.Printf("✓ Reverted [%s] %s", migration.Source, migration.FullName())

	return nil
}

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

func (m *migrator) executeTransactional(ctx context.Context, migrations []Migration) error {
	log.Printf("Executing %d migrations in single transaction...", len(migrations))

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

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

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migrations: %w", err)
	}

	for _, migration := range migrations {
		if err := m.tracker.RecordMigration(ctx, migration, 0); err != nil {
			log.Printf("Warning: failed to record migration %s: %v", migration.FullName(), err)
		}
	}

	log.Printf("✓ All %d migrations applied successfully", len(migrations))

	return nil
}

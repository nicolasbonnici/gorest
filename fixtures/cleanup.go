package fixtures

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/nicolasbonnici/gorest/crud"
)

type CleanupStrategy int

const (
	CleanupDelete CleanupStrategy = iota
	CleanupTruncate
	CleanupRollback
)

// Cleanup removes all loaded fixtures if cleanup is enabled.
func Cleanup(loader *Loader) error {
	return CleanupWithStrategy(loader, CleanupDelete)
}

func CleanupWithStrategy(loader *Loader, strategy CleanupStrategy) error {
	if !loader.ShouldCleanup() {
		return nil
	}

	switch strategy {
	case CleanupRollback:
		return cleanupWithRollback(loader)
	case CleanupTruncate:
		return cleanupWithTruncate(loader)
	case CleanupDelete:
		return cleanupWithDelete(loader)
	default:
		return fmt.Errorf("unknown cleanup strategy: %d (use CleanupDelete, CleanupTruncate, or CleanupRollback)", strategy)
	}
}

func cleanupWithRollback(loader *Loader) error {
	if loader.tx == nil {
		return fmt.Errorf("rollback strategy requires a transaction (call WithTransaction() before loading fixtures)")
	}

	if err := loader.Rollback(); err != nil {
		return fmt.Errorf("rollback cleanup failed: %w", err)
	}

	loader.mu.Lock()
	loader.loaded = make(map[string][]interface{})
	loader.mu.Unlock()
	return nil
}

func cleanupWithTruncate(loader *Loader) error {
	tables := make(map[string]bool)

	loader.mu.RLock()
	for _, fixtures := range loader.loaded {
		if len(fixtures) == 0 {
			continue
		}

		tableName := getTableName(fixtures[0])
		if tableName != "" {
			tables[tableName] = true
		}
	}
	loader.mu.RUnlock()

	dialect := loader.db.Dialect()
	driverName := loader.db.DriverName()

	for table := range tables {
		var query string
		quotedTable := dialect.QuoteIdentifier(table)
		if driverName == "sqlite" {
			query = fmt.Sprintf("DELETE FROM %s", quotedTable)
		} else {
			query = fmt.Sprintf("TRUNCATE TABLE %s", quotedTable)
		}

		var err error
		if loader.tx != nil {
			_, err = loader.tx.Exec(loader.ctx, query)
		} else {
			_, err = loader.db.Exec(loader.ctx, query)
		}

		if err != nil {
			return fmt.Errorf("failed to truncate table %s: %w (check foreign key constraints and table permissions)", table, err)
		}
	}

	loader.mu.Lock()
	loader.loaded = make(map[string][]interface{})
	loader.mu.Unlock()
	return nil
}

func cleanupWithDelete(loader *Loader) error {
	names := loader.GetLoadedFixtures()

	for _, name := range names {
		fixtures, ok := loader.Get(name)
		if !ok || len(fixtures) == 0 {
			continue
		}

		if err := deleteFixtures(loader, fixtures); err != nil {
			return fmt.Errorf("failed to delete fixtures %s: %w (use CleanupOrdered() for foreign key constraints)", name, err)
		}
	}

	loader.mu.Lock()
	loader.loaded = make(map[string][]interface{})
	loader.mu.Unlock()
	return nil
}

func deleteFixtures(loader *Loader, fixtures []interface{}) error {
	if len(fixtures) == 0 {
		return nil
	}

	tableName := getTableName(fixtures[0])
	if tableName == "" {
		return fmt.Errorf("unable to determine table name for fixture type %T (model must implement TableName())", fixtures[0])
	}

	ids := make([]interface{}, 0, len(fixtures))
	for _, f := range fixtures {
		id := getIDValue(f)
		if id != nil {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return nil
	}

	dialect := loader.db.Dialect()
	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = dialect.Placeholder(i + 1)
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s IN (%s)",
		dialect.QuoteIdentifier(tableName),
		dialect.QuoteIdentifier("id"),
		strings.Join(placeholders, ", "),
	)

	var err error
	if loader.tx != nil {
		_, err = loader.tx.Exec(loader.ctx, query, ids...)
	} else {
		_, err = loader.db.Exec(loader.ctx, query, ids...)
	}

	if err != nil {
		return fmt.Errorf("failed to delete from %s: %w (check foreign key constraints)", tableName, err)
	}

	return nil
}

func getTableName(model interface{}) string {
	if m, ok := model.(crud.Model); ok {
		return m.TableName()
	}

	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	method := val.MethodByName("TableName")
	if method.IsValid() {
		results := method.Call(nil)
		if len(results) > 0 {
			if tableName, ok := results[0].Interface().(string); ok {
				return tableName
			}
		}
	}

	return ""
}

func getIDValue(model interface{}) interface{} {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("db")
		if tag == "id" {
			idVal := val.Field(i)
			if idVal.IsValid() && !idVal.IsZero() {
				return idVal.Interface()
			}
		}
	}

	return nil
}

// CleanupOrdered cleans up fixtures in specified order (child to parent tables for foreign keys).
func CleanupOrdered(loader *Loader, order []string) error {
	if !loader.ShouldCleanup() {
		return nil
	}

	for i := len(order) - 1; i >= 0; i-- {
		name := order[i]
		fixtures, ok := loader.Get(name)
		if !ok || len(fixtures) == 0 {
			continue
		}

		if err := deleteFixtures(loader, fixtures); err != nil {
			return fmt.Errorf("failed to delete fixtures %s: %w (delete child tables before parent tables)", name, err)
		}
	}

	loader.mu.Lock()
	loader.loaded = make(map[string][]interface{})
	loader.mu.Unlock()
	return nil
}

package fixtures

import (
	"fmt"
	"reflect"

	"github.com/nicolasbonnici/gorest/crud"
)

type CleanupStrategy int

const (
	CleanupDelete CleanupStrategy = iota
	// CleanupTruncate truncates all tables (faster but requires permissions)
	CleanupTruncate
	// CleanupRollback rolls back the transaction (only works with transactions)
	CleanupRollback
)

// Cleanup removes all loaded fixtures from the database.
// This should be called in a defer or t.Cleanup() to ensure fixtures are removed after tests.
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
		return fmt.Errorf("unknown cleanup strategy: %d", strategy)
	}
}

func cleanupWithRollback(loader *Loader) error {
	if loader.tx == nil {
		return fmt.Errorf("rollback strategy requires a transaction")
	}

	if err := loader.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	loader.loaded = make(map[string][]interface{})
	return nil
}

func cleanupWithTruncate(loader *Loader) error {
	tables := make(map[string]bool)

	for _, fixtures := range loader.loaded {
		if len(fixtures) == 0 {
			continue
		}

		tableName := getTableName(fixtures[0])
		if tableName != "" {
			tables[tableName] = true
		}
	}

	for table := range tables {
		var query string
		switch loader.db.Dialect().QuoteIdentifier("test") {
		case `"test"`:
			query = fmt.Sprintf("DELETE FROM %s", table)
		default:
			query = fmt.Sprintf("TRUNCATE TABLE %s", table)
		}

		var err error
		if loader.tx != nil {
			_, err = loader.tx.Exec(loader.ctx, query)
		} else {
			_, err = loader.db.Exec(loader.ctx, query)
		}

		if err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	loader.loaded = make(map[string][]interface{})
	return nil
}

func cleanupWithDelete(loader *Loader) error {
	names := loader.GetLoadedFixtures()

	for i := len(names) - 1; i >= 0; i-- {
		name := names[i]
		fixtures, ok := loader.Get(name)
		if !ok || len(fixtures) == 0 {
			continue
		}

		if err := deleteFixtures(loader, fixtures); err != nil {
			return fmt.Errorf("failed to delete fixtures %s: %w", name, err)
		}
	}

	loader.loaded = make(map[string][]interface{})
	return nil
}

func deleteFixtures(loader *Loader, fixtures []interface{}) error {
	if len(fixtures) == 0 {
		return nil
	}

	tableName := getTableName(fixtures[0])
	if tableName == "" {
		return fmt.Errorf("unable to determine table name for fixture type %T", fixtures[0])
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

	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE id IN (%s)",
		tableName,
		joinStrings(placeholders, ", "),
	)

	var err error
	if loader.tx != nil {
		_, err = loader.tx.Exec(loader.ctx, query, ids...)
	} else {
		_, err = loader.db.Exec(loader.ctx, query, ids...)
	}

	if err != nil {
		return fmt.Errorf("failed to delete from %s: %w", tableName, err)
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

// CleanupOrdered performs cleanup in a specific order to respect foreign key constraints.
// The order should be from child tables to parent tables (reverse dependency order).
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
			return fmt.Errorf("failed to delete fixtures %s: %w", name, err)
		}
	}

	loader.loaded = make(map[string][]interface{})
	return nil
}

// Package fixtures provides test data management for database-driven applications.
// Supports programmatic and file-based fixtures with automatic cleanup and transaction support.
package fixtures

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
	"gopkg.in/yaml.v3"
)

// Loader handles loading fixtures into a database.
// Thread-safe for concurrent goroutine access. For tests, use separate instances per test for isolation.
type Loader struct {
	db      database.Database
	tx      database.Tx
	loaded  map[string][]interface{}
	mu      sync.RWMutex
	cleanup bool
	ctx     context.Context
}

func New(db database.Database) *Loader {
	return &Loader{
		db:      db,
		loaded:  make(map[string][]interface{}),
		cleanup: false,
		ctx:     context.Background(),
	}
}

func (l *Loader) WithContext(ctx context.Context) *Loader {
	l.ctx = ctx
	return l
}

// WithTransaction enables transaction support for test isolation
func (l *Loader) WithTransaction() (*Loader, error) {
	if l.tx != nil {
		return l, fmt.Errorf("transaction already active (commit or rollback before starting a new one)")
	}

	tx, err := l.db.Begin(l.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w (check database connection and permissions)", err)
	}

	l.tx = tx
	return l, nil
}

func (l *Loader) Commit() error {
	if l.tx == nil {
		return fmt.Errorf("no transaction to commit (call WithTransaction() first)")
	}

	if err := l.tx.Commit(l.ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w (data may be partially committed)", err)
	}

	l.tx = nil
	return nil
}

func (l *Loader) Rollback() error {
	if l.tx == nil {
		return fmt.Errorf("no transaction to rollback (call WithTransaction() first)")
	}

	if err := l.tx.Rollback(l.ctx); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w (transaction may be in inconsistent state)", err)
	}

	l.tx = nil
	return nil
}

// Load loads fixtures from Go structs. Use WithTransaction() for atomic all-or-nothing loading.
// Without a transaction, partial failures leave some fixtures in the database.
func Load[T crud.Model](l *Loader, name string, fixtures []T) (*Loader, error) {
	var loaded []interface{}

	if len(fixtures) > 0 {
		for i, fixture := range fixtures {
			if err := l.insertFixture(fixture); err != nil {
				return l, fmt.Errorf("failed to create fixture %s[%d]: %w (hint: use WithTransaction() for atomic loading)", name, i, err)
			}
			loaded = append(loaded, fixture)
		}
	}

	l.mu.Lock()
	l.loaded[name] = loaded
	l.mu.Unlock()
	return l, nil
}

func (l *Loader) insertFixture(m crud.Model) error {
	v := reflect.ValueOf(m)
	typ := reflect.TypeOf(m)
	dialect := l.db.Dialect()

	var cols []string
	var vals []interface{}
	var placeholders []string

	paramIndex := 1
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("db")
		if tag == "" || tag == "created_at" || tag == "updated_at" {
			continue
		}
		fieldVal := v.Field(i)
		if tag == "id" && fieldVal.IsZero() {
			continue
		}
		cols = append(cols, dialect.QuoteIdentifier(tag))
		vals = append(vals, fieldVal.Interface())
		placeholders = append(placeholders, dialect.Placeholder(paramIndex))
		paramIndex++
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		dialect.QuoteIdentifier(m.TableName()),
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	var err error
	if l.tx != nil {
		_, err = l.tx.Exec(l.ctx, query, vals...)
	} else {
		_, err = l.db.Exec(l.ctx, query, vals...)
	}

	return err
}

func (l *Loader) LoadFromYAML(name string, filePath string, target interface{}) (*Loader, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return l, fmt.Errorf("failed to read YAML file %s: %w (check file path and permissions)", filePath, err)
	}

	if err := yaml.Unmarshal(data, target); err != nil {
		return l, fmt.Errorf("failed to unmarshal YAML from %s: %w (check YAML syntax and target type)", filePath, err)
	}

	if err := l.insertRawData(name, target); err != nil {
		return l, err
	}

	return l, nil
}

func (l *Loader) LoadFromJSON(name string, filePath string, target interface{}) (*Loader, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return l, fmt.Errorf("failed to read JSON file %s: %w (check file path and permissions)", filePath, err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return l, fmt.Errorf("failed to unmarshal JSON from %s: %w (check JSON syntax and target type)", filePath, err)
	}

	if err := l.insertRawData(name, target); err != nil {
		return l, err
	}

	return l, nil
}

func (l *Loader) insertRawData(name string, data interface{}) error {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Slice {
		return fmt.Errorf("target must be a slice, got %s (pass &[]YourModel{} as target)", val.Kind())
	}

	loaded := make([]interface{}, val.Len())
	for i := 0; i < val.Len(); i++ {
		loaded[i] = val.Index(i).Interface()
	}

	l.mu.Lock()
	l.loaded[name] = loaded
	l.mu.Unlock()
	return nil
}

func (l *Loader) Get(name string) ([]interface{}, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	fixtures, ok := l.loaded[name]
	return fixtures, ok
}

func GetTyped[T any](l *Loader, name string) ([]T, error) {
	fixtures, ok := l.Get(name)
	if !ok {
		return nil, fmt.Errorf("fixtures not found: %s (load fixtures with this name first)", name)
	}

	result := make([]T, 0, len(fixtures))
	for i, f := range fixtures {
		typed, ok := f.(T)
		if !ok {
			return nil, fmt.Errorf("fixture %s[%d] is not of type %T (got %T, check fixture type matches expected type)", name, i, *new(T), f)
		}
		result = append(result, typed)
	}

	return result, nil
}

func (l *Loader) EnableCleanup() *Loader {
	l.cleanup = true
	return l
}

func (l *Loader) ShouldCleanup() bool {
	return l.cleanup
}

func (l *Loader) GetLoadedFixtures() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	names := make([]string, len(l.loaded))
	i := 0
	for name := range l.loaded {
		names[i] = name
		i++
	}
	return names
}

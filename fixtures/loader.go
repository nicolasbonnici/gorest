package fixtures

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
	"gopkg.in/yaml.v3"
)

// Loader handles loading fixtures into a database
type Loader struct {
	db      database.Database
	tx      database.Tx
	loaded  map[string][]interface{}
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
		return l, fmt.Errorf("transaction already started")
	}

	tx, err := l.db.Begin(l.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	l.tx = tx
	return l, nil
}

func (l *Loader) Commit() error {
	if l.tx == nil {
		return fmt.Errorf("no transaction to commit")
	}

	if err := l.tx.Commit(l.ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	l.tx = nil
	return nil
}

func (l *Loader) Rollback() error {
	if l.tx == nil {
		return fmt.Errorf("no transaction to rollback")
	}

	if err := l.tx.Rollback(l.ctx); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	l.tx = nil
	return nil
}

// Load loads fixtures from Go structs using CRUD operations
func Load[T crud.Model](l *Loader, name string, fixtures []T) (*Loader, error) {
	var loaded []interface{}

	if len(fixtures) > 0 {
		for i, fixture := range fixtures {
			if err := l.insertFixture(fixture); err != nil {
				return l, fmt.Errorf("failed to create fixture %s[%d]: %w", name, i, err)
			}
			loaded = append(loaded, fixture)
		}
	}

	l.loaded[name] = loaded
	return l, nil
}

// insertFixture inserts a single fixture, preserving IDs if present
func (l *Loader) insertFixture(m crud.Model) error {
	v := reflect.ValueOf(m)
	typ := reflect.TypeOf(m)

	var cols []string
	var vals []interface{}
	var placeholders []string

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
		cols = append(cols, tag)
		vals = append(vals, fieldVal.Interface())
		placeholders = append(placeholders, "?")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		m.TableName(),
		joinStrings(cols, ", "),
		joinStrings(placeholders, ", "),
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
		return l, fmt.Errorf("failed to read YAML file %s: %w", filePath, err)
	}

	if err := yaml.Unmarshal(data, target); err != nil {
		return l, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	if err := l.insertRawData(name, target); err != nil {
		return l, err
	}

	return l, nil
}

func (l *Loader) LoadFromJSON(name string, filePath string, target interface{}) (*Loader, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return l, fmt.Errorf("failed to read JSON file %s: %w", filePath, err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return l, fmt.Errorf("failed to unmarshal JSON: %w", err)
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
		return fmt.Errorf("target must be a slice, got %s", val.Kind())
	}

	loaded := make([]interface{}, val.Len())
	for i := 0; i < val.Len(); i++ {
		loaded[i] = val.Index(i).Interface()
	}

	l.loaded[name] = loaded
	return nil
}

func (l *Loader) Get(name string) ([]interface{}, bool) {
	fixtures, ok := l.loaded[name]
	return fixtures, ok
}

// GetTyped retrieves loaded fixtures by name with type assertion
func GetTyped[T any](l *Loader, name string) ([]T, error) {
	fixtures, ok := l.Get(name)
	if !ok {
		return nil, fmt.Errorf("fixtures not found: %s", name)
	}

	result := make([]T, 0, len(fixtures))
	for i, f := range fixtures {
		typed, ok := f.(T)
		if !ok {
			return nil, fmt.Errorf("fixture %s[%d] is not of type %T", name, i, *new(T))
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
	names := make([]string, 0, len(l.loaded))
	for name := range l.loaded {
		names = append(names, name)
	}
	return names
}

// TxCRUD wraps a transaction to implement the crud.Repository interface
type TxCRUD[T crud.Model] struct {
	tx database.Tx
}

func NewTxCRUD[T crud.Model](tx database.Tx) *TxCRUD[T] {
	return &TxCRUD[T]{tx: tx}
}

// Create implements crud.Repository.Create using the transaction
func (t *TxCRUD[T]) Create(ctx context.Context, m T) error {
	v := reflect.ValueOf(m)
	typ := reflect.TypeOf(m)

	var cols []string
	var vals []interface{}
	var placeholders []string

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
		cols = append(cols, tag)
		vals = append(vals, fieldVal.Interface())
		placeholders = append(placeholders, "?")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		m.TableName(),
		joinStrings(cols, ", "),
		joinStrings(placeholders, ", "),
	)

	_, err := t.tx.Exec(ctx, query, vals...)
	return err
}

// GetAll implements crud.Repository.GetAll
func (t *TxCRUD[T]) GetAll(ctx context.Context) ([]T, error) {
	return nil, fmt.Errorf("GetAll not implemented for TxCRUD")
}

// GetByID implements crud.Repository.GetByID
func (t *TxCRUD[T]) GetByID(ctx context.Context, id any) (*T, error) {
	return nil, fmt.Errorf("GetByID not implemented for TxCRUD")
}

// Update implements crud.Repository.Update
func (t *TxCRUD[T]) Update(ctx context.Context, id any, m T) error {
	return fmt.Errorf("Update not implemented for TxCRUD")
}

// Delete implements crud.Repository.Delete
func (t *TxCRUD[T]) Delete(ctx context.Context, id any) error {
	return fmt.Errorf("Delete not implemented for TxCRUD")
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

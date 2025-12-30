package fixtures

import (
	"context"
	"fmt"
	"testing"

	"github.com/nicolasbonnici/gorest/database"
)

// Builder provides a fluent API for loading and managing fixtures
type Builder struct {
	loader *Loader
	t      *testing.T
	err    error
}

// NewBuilder creates a new Builder with the given database
func NewBuilder(db database.Database) *Builder {
	return &Builder{
		loader: New(db),
	}
}

// NewBuilderWithT creates a new Builder that integrates with testing.T
// This will automatically fail the test if any fixture operation fails
func NewBuilderWithT(t *testing.T, db database.Database) *Builder {
	t.Helper()
	return &Builder{
		loader: New(db),
		t:      t,
	}
}

// WithContext sets the context for database operations
func (b *Builder) WithContext(ctx context.Context) *Builder {
	b.loader.WithContext(ctx)
	return b
}

// WithTransaction enables transaction support for test isolation
// The transaction must be committed or rolled back manually
func (b *Builder) WithTransaction() *Builder {
	if b.err != nil {
		return b
	}

	_, err := b.loader.WithTransaction()
	if err != nil {
		b.err = err
		if b.t != nil {
			b.t.Fatalf("failed to start transaction: %v", err)
		}
	}
	return b
}

// Load loads fixtures from Go structs using CRUD operations
func (b *Builder) Load(name string, fixtures interface{}) *Builder {
	if b.err != nil {
		return b
	}

	err := b.loadTyped(name, fixtures)
	if err != nil {
		b.err = err
		if b.t != nil {
			b.t.Fatalf("failed to load fixtures %s: %v", name, err)
		}
	}
	return b
}

// loadTyped handles type-safe loading with reflection
func (b *Builder) loadTyped(name string, fixtures interface{}) error {
	switch v := fixtures.(type) {
	case []User:
		_, err := Load(b.loader, name, v)
		return err
	case []Todo:
		_, err := Load(b.loader, name, v)
		return err
	default:
		return fmt.Errorf("unsupported fixture type: %T", fixtures)
	}
}

// LoadFromYAML loads fixtures from a YAML file
func (b *Builder) LoadFromYAML(name string, filePath string, target interface{}) *Builder {
	if b.err != nil {
		return b
	}

	_, err := b.loader.LoadFromYAML(name, filePath, target)
	if err != nil {
		b.err = err
		if b.t != nil {
			b.t.Fatalf("failed to load fixtures from YAML %s: %v", filePath, err)
		}
	}
	return b
}

// LoadFromJSON loads fixtures from a JSON file
func (b *Builder) LoadFromJSON(name string, filePath string, target interface{}) *Builder {
	if b.err != nil {
		return b
	}

	_, err := b.loader.LoadFromJSON(name, filePath, target)
	if err != nil {
		b.err = err
		if b.t != nil {
			b.t.Fatalf("failed to load fixtures from JSON %s: %v", filePath, err)
		}
	}
	return b
}

// Commit commits the current transaction if one exists
func (b *Builder) Commit() *Builder {
	if b.err != nil {
		return b
	}

	err := b.loader.Commit()
	if err != nil {
		b.err = err
		if b.t != nil {
			b.t.Fatalf("failed to commit transaction: %v", err)
		}
	}
	return b
}

// Rollback rolls back the current transaction if one exists
func (b *Builder) Rollback() *Builder {
	if b.err != nil {
		return b
	}

	err := b.loader.Rollback()
	if err != nil {
		b.err = err
		if b.t != nil {
			b.t.Fatalf("failed to rollback transaction: %v", err)
		}
	}
	return b
}

// Cleanup enables automatic cleanup of loaded fixtures
func (b *Builder) Cleanup() *Builder {
	if b.err != nil {
		return b
	}

	b.loader.EnableCleanup()

	if b.t != nil {
		b.t.Cleanup(func() {
			if err := Cleanup(b.loader); err != nil {
				b.t.Errorf("failed to cleanup fixtures: %v", err)
			}
		})
	}

	return b
}

// EnableCleanup enables automatic cleanup of loaded fixtures (alias for Cleanup)
func (b *Builder) EnableCleanup() *Builder {
	return b.Cleanup()
}

// Get retrieves loaded fixtures by name
func (b *Builder) Get(name string) ([]interface{}, bool) {
	return b.loader.Get(name)
}

// GetTyped retrieves loaded fixtures by name with type assertion
func (b *Builder) GetTyped(name string, target interface{}) error {
	if b.err != nil {
		return b.err
	}

	switch t := target.(type) {
	case *[]User:
		users, err := GetTyped[User](b.loader, name)
		if err != nil {
			return err
		}
		*t = users
		return nil
	case *[]Todo:
		todos, err := GetTyped[Todo](b.loader, name)
		if err != nil {
			return err
		}
		*t = todos
		return nil
	default:
		return fmt.Errorf("unsupported target type: %T", target)
	}
}

// Error returns any error that occurred during fixture operations
func (b *Builder) Error() error {
	return b.err
}

// Loader returns the underlying Loader instance
func (b *Builder) Loader() *Loader {
	return b.loader
}

// User is a test model for fixtures
type User struct {
	ID        string `db:"id"`
	Firstname string `db:"firstname"`
	Lastname  string `db:"lastname"`
	Email     string `db:"email"`
	Password  string `db:"password"`
	UpdatedAt string `db:"updated_at"`
	CreatedAt string `db:"created_at"`
}

func (u User) TableName() string {
	return "users"
}

// Todo is a test model for fixtures
type Todo struct {
	ID        string `db:"id"`
	UserID    string `db:"user_id"`
	Title     string `db:"title"`
	Content   string `db:"content"`
	UpdatedAt string `db:"updated_at"`
	CreatedAt string `db:"created_at"`
}

func (t Todo) TableName() string {
	return "todo"
}

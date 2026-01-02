package fixtures

import (
	"context"
	"testing"

	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
)

// Builder provides a fluent API for loading and managing fixtures
type Builder struct {
	loader *Loader
	t      *testing.T
	err    error
}

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

// LoadBuilder is a generic function for loading fixtures from Go structs using CRUD operations.
// It provides a type-safe way to load fixtures and returns the builder for method chaining.
//
// IMPORTANT: For atomic loading (all-or-nothing), use WithTransaction():
//
//	LoadBuilder(builder.WithTransaction(), "users", users)
//	LoadBuilder(builder, "todos", todos)
//	builder.Commit()
//
// Without a transaction, partial failures will leave some fixtures in the database.
func LoadBuilder[T crud.Model](b *Builder, name string, fixtures []T) *Builder {
	if b.err != nil {
		return b
	}

	_, err := Load(b.loader, name, fixtures)
	if err != nil {
		b.err = err
		if b.t != nil {
			b.t.Fatalf("failed to load fixtures %s: %v", name, err)
		}
	}
	return b
}

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

func (b *Builder) Get(name string) ([]interface{}, bool) {
	return b.loader.Get(name)
}

// GetTyped retrieves loaded fixtures by name with type assertion.
// Returns the typed slice of fixtures and an error if the fixtures are not found
// or cannot be cast to the requested type.
func GetTypedFromBuilder[T any](b *Builder, name string) ([]T, error) {
	if b.err != nil {
		return nil, b.err
	}

	return GetTyped[T](b.loader, name)
}

func (b *Builder) Error() error {
	return b.err
}

func (b *Builder) Loader() *Loader {
	return b.loader
}

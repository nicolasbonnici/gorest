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
			b.t.Fatalf("transaction start failed: %v (check database connection)", err)
		}
	}
	return b
}

// LoadBuilder loads fixtures from Go structs using the builder pattern.
// Use WithTransaction() for atomic all-or-nothing loading.
func LoadBuilder[T crud.Model](b *Builder, name string, fixtures []T) *Builder {
	if b.err != nil {
		return b
	}

	_, err := Load(b.loader, name, fixtures)
	if err != nil {
		b.err = err
		if b.t != nil {
			b.t.Fatalf("fixture load failed for %s: %v (check table exists and model implements TableName())", name, err)
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
			b.t.Fatalf("YAML fixture load failed for %s: %v", filePath, err)
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
			b.t.Fatalf("JSON fixture load failed for %s: %v", filePath, err)
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
			b.t.Fatalf("transaction commit failed: %v (did you call WithTransaction()?)", err)
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
			b.t.Fatalf("transaction rollback failed: %v (did you call WithTransaction()?)", err)
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
				b.t.Errorf("fixture cleanup failed: %v (check database state and foreign key constraints)", err)
			}
		})
	}

	return b
}


func (b *Builder) Get(name string) ([]interface{}, bool) {
	return b.loader.Get(name)
}

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

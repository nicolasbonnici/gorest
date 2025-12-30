//go:build integration

package sqlite_test

import (
	"context"
	"testing"

	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
	"github.com/nicolasbonnici/gorest/internal/testhelpers"
)

func TestSQLiteIntrospector_GetRelations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	introspector := db.Introspector()
	ctx := context.Background()

	relations, err := introspector.GetRelations(ctx)
	if err != nil {
		t.Fatalf("Failed to get relations: %v", err)
	}

	// Should have at least one foreign key (todo -> users)
	if len(relations) == 0 {
		t.Error("Expected at least one relation in schema")
	}

	// Find the user_id foreign key from todo table
	var userRelation *database.Relation
	for i := range relations {
		if relations[i].ChildTable == "todo" &&
			relations[i].ChildColumn == "user_id" &&
			relations[i].ParentTable == "users" {
			userRelation = &relations[i]
			break
		}
	}

	if userRelation == nil {
		t.Error("Expected todo table to have foreign key to users table")
	} else {
		if userRelation.ParentColumn != "id" {
			t.Errorf("Expected foreign key to reference users.id, got %s", userRelation.ParentColumn)
		}
	}
}

func TestSQLiteIntrospector_LoadSchema(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	introspector := db.Introspector()
	ctx := context.Background()

	schema, err := introspector.LoadSchema(ctx)
	if err != nil {
		t.Fatalf("Failed to load schema: %v", err)
	}

	if len(schema) == 0 {
		t.Error("Expected at least one table in schema")
	}

	// Find users table
	var usersTable *database.TableSchema
	for i := range schema {
		if schema[i].TableName == "users" {
			usersTable = &schema[i]
			break
		}
	}

	if usersTable == nil {
		t.Fatal("Expected users table in schema")
	}

	// Verify table has columns
	if len(usersTable.Columns) == 0 {
		t.Error("Expected users table to have columns")
	}
}

func TestSQLiteIntrospector_GetColumns(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	introspector := db.Introspector()
	ctx := context.Background()

	columns, err := introspector.GetColumns(ctx, "users")
	if err != nil {
		t.Fatalf("Failed to get columns: %v", err)
	}

	if len(columns) == 0 {
		t.Error("Expected users table to have columns")
	}

	// Find email column
	var emailCol *database.Column
	for i := range columns {
		if columns[i].Name == "email" {
			emailCol = &columns[i]
			break
		}
	}

	if emailCol == nil {
		t.Fatal("Expected email column in users table")
	}

	if emailCol.Type == "" {
		t.Error("Expected email column to have a type")
	}
}

func setupTestDB(t *testing.T) database.Database {
	t.Helper()

	schema := `
CREATE TABLE users (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    firstname TEXT NOT NULL,
    lastname TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password TEXT,
    updated_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE todo (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    user_id TEXT,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    updated_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);`

	return testhelpers.SetupSQLiteFileWithSchema(t, schema)
}

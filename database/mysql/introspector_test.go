//go:build integration

package mysql_test

import (
	"context"
	"testing"

	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
	"github.com/nicolasbonnici/gorest/internal/testhelpers"
)

func TestMySQLIntrospector_GetColumns(t *testing.T) {
	db := setupTestDB(t)

	introspector := db.Introspector()
	ctx := context.Background()

	columns, err := introspector.GetColumns(ctx, "users")
	if err != nil {
		t.Fatalf("Failed to get columns: %v", err)
	}

	if len(columns) == 0 {
		t.Error("Expected users table to have columns")
	}

	// Find email column and verify its properties
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

	// Find id column
	var idCol *database.Column
	for i := range columns {
		if columns[i].Name == "id" {
			idCol = &columns[i]
			break
		}
	}

	if idCol == nil {
		t.Fatal("Expected id column in users table")
	}
}

func TestMySQLIntrospector_GetRelations(t *testing.T) {
	db := setupTestDB(t)

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

func setupTestDB(t *testing.T) database.Database {
	t.Helper()
	return testhelpers.SetupMySQLWithDSN(t, "testuser:testpass@tcp(localhost:3307)/mydb_test")
}

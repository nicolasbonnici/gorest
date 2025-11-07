//go:build integration

package mysql_test

import (
	"context"
	"testing"
	"time"

	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/mysql"
)

func TestMySQLIntrospector_GetColumns(t *testing.T) {
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

func setupTestDB(t *testing.T) database.Database {
	t.Helper()

	dsn := "testuser:testpass@tcp(localhost:3307)/mydb_test"
	db, err := database.Open("mysql", dsn)
	if err != nil {
		t.Skipf("MySQL not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Skipf("MySQL ping failed: %v", err)
	}

	return db
}

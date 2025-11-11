//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/postgres"
)

func TestPostgresIntrospector_LoadSchema(t *testing.T) {
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

	// Check for expected columns
	expectedColumns := map[string]bool{
		"id":         false,
		"firstname":  false,
		"lastname":   false,
		"email":      false,
		"password":   false,
		"created_at": false,
		"updated_at": false,
	}

	for _, col := range usersTable.Columns {
		if _, ok := expectedColumns[col.Name]; ok {
			expectedColumns[col.Name] = true
		}
	}

	for col, found := range expectedColumns {
		if !found {
			t.Errorf("Expected column %s not found in users table", col)
		}
	}

	// Find todo table
	var todoTable *database.TableSchema
	for i := range schema {
		if schema[i].TableName == "todo" {
			todoTable = &schema[i]
			break
		}
	}

	if todoTable == nil {
		t.Fatal("Expected todo table in schema")
	}

	if len(todoTable.Columns) == 0 {
		t.Error("Expected todo table to have columns")
	}
}

func TestPostgresIntrospector_GetColumns(t *testing.T) {
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

func TestPostgresIntrospector_GetRelations(t *testing.T) {
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

	dsn := "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable"
	db, err := database.Open("postgres", dsn)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Skipf("PostgreSQL ping failed: %v", err)
	}

	return db
}

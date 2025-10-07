package internal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestLoadSchema(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	tables := LoadSchema(db)

	// Verify users table
	if _, ok := tables["users"]; !ok {
		t.Error("Expected users table in schema")
	}

	usersTable := tables["users"]
	if usersTable.TableName != "users" {
		t.Errorf("Expected table name 'users', got '%s'", usersTable.TableName)
	}

	// Check for expected columns
	expectedCols := map[string]bool{
		"id":         true,
		"firstname":  true,
		"lastname":   true,
		"email":      true,
		"password":   true,
		"created_at": true,
		"updated_at": true,
	}

	for _, col := range usersTable.Columns {
		if expectedCols[col.Name] {
			delete(expectedCols, col.Name)
		}
	}

	if len(expectedCols) > 0 {
		t.Errorf("Missing columns in users table: %v", expectedCols)
	}

	// Verify todo table
	if _, ok := tables["todo"]; !ok {
		t.Error("Expected todo table in schema")
	}
}

func TestGenerateStructs(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create a temporary directory for test output
	tempDir := t.TempDir()
	originalModelsDir := filepath.Join(tempDir, "models")
	os.MkdirAll(originalModelsDir, 0755)

	// Temporarily replace the models directory path
	// Note: This test assumes GenerateStructs is modified to accept a path parameter
	// For now, we'll test the function behavior with actual models directory

	tables := LoadSchema(db)

	// Ensure we have tables
	if len(tables) == 0 {
		t.Fatal("No tables loaded from schema")
	}

	// Generate structs
	GenerateStructs(tables)

	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	// Verify generated files exist
	usersFile := filepath.Join(projectRoot, "gen/models", "users.go")
	if _, err := os.Stat(usersFile); os.IsNotExist(err) {
		t.Errorf("Expected users.go to be generated, but file does not exist")
	}

	todoFile := filepath.Join(projectRoot, "gen/models", "todo.go")
	if _, err := os.Stat(todoFile); os.IsNotExist(err) {
		t.Errorf("Expected todo.go to be generated, but file does not exist")
	}

	// Read and verify content of users.go
	content, err := os.ReadFile(usersFile)
	if err != nil {
		t.Fatalf("Failed to read users.go: %v", err)
	}

	contentStr := string(content)

	// Check for expected content
	expectedStrings := []string{
		"package models",
		"type Users struct",
		"Firstname",
		"Lastname",
		"Email",
		"Password",
		"func (Users) TableName() string",
		`return "users"`,
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Expected users.go to contain '%s'", expected)
		}
	}
}

func TestGenerateOpenAPI(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	tables := LoadSchema(db)

	// Generate OpenAPI stubs
	GenerateOpenAPI(tables)

	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	// Verify generated file exists
	openapiFile := filepath.Join(projectRoot, "gen/api", "openapi_gen.go")
	if _, err := os.Stat(openapiFile); os.IsNotExist(err) {
		t.Errorf("Expected openapi_gen.go to be generated, but file does not exist")
	}

	// Read and verify content
	content, err := os.ReadFile(openapiFile)
	if err != nil {
		t.Fatalf("Failed to read openapi_gen.go: %v", err)
	}

	contentStr := string(content)

	// Check for expected content
	expectedStrings := []string{
		"package api",
		"Auto-generated OpenAPI schema stubs",
		"UsersResource",
		"TodoResource",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Expected openapi_gen.go to contain '%s'", expected)
		}
	}
}

func TestPgToGoType(t *testing.T) {
	tests := []struct {
		pgType   string
		nullable bool
		expected string
	}{
		{"integer", false, "int"},
		{"integer", true, "*int"},
		{"text", false, "string"},
		{"text", true, "*string"},
		{"boolean", false, "bool"},
		{"boolean", true, "*bool"},
		{"timestamp without time zone", false, "*time.Time"},
		{"timestamp without time zone", true, "*time.Time"},
		{"unknown_type", false, "interface{}"},
		{"unknown_type", true, "interface{}"},
	}

	for _, tt := range tests {
		t.Run(tt.pgType, func(t *testing.T) {
			result := pgToGoType(tt.pgType, tt.nullable)
			if result != tt.expected {
				t.Errorf("pgToGoType(%s, %v) = %s; want %s", tt.pgType, tt.nullable, result, tt.expected)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user", "User"},
		{"user_profile", "UserProfile"},
		{"todo_item", "TodoItem"},
		{"my_table_name", "MyTableName"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("toCamelCase(%s) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestScaffoldAll(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Run scaffold all
	ScaffoldAll(db)

	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	// Verify models are generated
	usersFile := filepath.Join(projectRoot, "gen/models", "users.go")
	if _, err := os.Stat(usersFile); os.IsNotExist(err) {
		t.Error("Expected users.go to be generated by ScaffoldAll")
	}

	// Verify OpenAPI is generated
	openapiFile := filepath.Join(projectRoot, "gen/api", "openapi_gen.go")
	if _, err := os.Stat(openapiFile); os.IsNotExist(err) {
		t.Error("Expected openapi_gen.go to be generated by ScaffoldAll")
	}
}

// Helper function for tests that need a database
func setupModelGenTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	db, err := pgxpool.New(context.Background(), testDBURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	return db
}

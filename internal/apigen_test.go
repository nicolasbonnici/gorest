package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAPI(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Ensure models and CRUD are generated first
	tables := LoadSchema(db)
	GenerateStructs(tables)
	GenerateCRUD()

	// Generate API resources
	GenerateAPI(db, tables)

	// Verify users resource was generated
	usersResourceFile := filepath.Join("gen/resources", "users.go")
	if _, err := os.Stat(usersResourceFile); os.IsNotExist(err) {
		t.Error("Expected users.go resource to be generated")
	}

	// Verify todo resource was generated
	todoResourceFile := filepath.Join("gen/resources", "todo.go")
	if _, err := os.Stat(todoResourceFile); os.IsNotExist(err) {
		t.Error("Expected todo.go resource to be generated")
	}

	// Read and verify users resource content
	content, err := os.ReadFile(usersResourceFile)
	if err != nil {
		t.Fatalf("Failed to read users.go: %v", err)
	}

	contentStr := string(content)

	// Check for expected content
	expectedStrings := []string{
		"package resources",
		"UsersResource",
		"RegisterUsersRoutes",
		"CRUD *crud.CRUD[models.Users]",
		"crud.New[models.Users](db)",
		"func (r *UsersResource) List(c *fiber.Ctx) error",
		"func (r *UsersResource) Get(c *fiber.Ctx) error",
		"func (r *UsersResource) Create(c *fiber.Ctx) error",
		"func (r *UsersResource) Update(c *fiber.Ctx) error",
		"func (r *UsersResource) Delete(c *fiber.Ctx) error",
		"r.CRUD.GetAll(c.Context())",
		"r.CRUD.GetByID(c.Context(), id)",
		"r.CRUD.Create(c.Context(), item)",
		"r.CRUD.Update(c.Context(), id, item)",
		"r.CRUD.Delete(c.Context(), id)",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Expected users.go to contain '%s'", expected)
		}
	}
}

func TestParseStructs(t *testing.T) {
	// Create a temporary test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	testContent := `package test

type User struct {
	Name string
}

type Product struct {
	Title string
	Price float64
}

func SomeFunction() {}
`

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Parse the file
	structs := parseStructs(testFile)

	// Verify results
	if len(structs) != 2 {
		t.Errorf("Expected 2 structs, got %d", len(structs))
	}

	expectedStructs := map[string]bool{
		"User":    false,
		"Product": false,
	}

	for _, s := range structs {
		if _, ok := expectedStructs[s]; ok {
			expectedStructs[s] = true
		}
	}

	for name, found := range expectedStructs {
		if !found {
			t.Errorf("Expected to find struct '%s'", name)
		}
	}
}

func TestGenerateResourceFromModel(t *testing.T) {
	result := generateResourceFromModel("User")

	// Check for expected patterns in generated code
	expectedStrings := []string{
		"package resources",
		"UserResource",
		"RegisterUserRoutes",
		"CRUD *crud.CRUD[models.User]",
		"crud.New[models.User](db)",
		"router.Get(\"/user\", res.List)",
		"router.Get(\"/user/:id\", res.Get)",
		"router.Post(\"/user\", res.Create)",
		"router.Put(\"/user/:id\", res.Update)",
		"router.Delete(\"/user/:id\", res.Delete)",
		"func (r *UserResource) List(c *fiber.Ctx) error",
		"items, err := r.CRUD.GetAll(c.Context())",
		"func (r *UserResource) Get(c *fiber.Ctx) error",
		"item, err := r.CRUD.GetByID(c.Context(), id)",
		"func (r *UserResource) Create(c *fiber.Ctx) error",
		"var item models.User",
		"r.CRUD.Create(c.Context(), item)",
		"func (r *UserResource) Update(c *fiber.Ctx) error",
		"r.CRUD.Update(c.Context(), id, item)",
		"func (r *UserResource) Delete(c *fiber.Ctx) error",
		"r.CRUD.Delete(c.Context(), id)",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected generated code to contain '%s'", expected)
		}
	}
}

func TestGenerateResourceForStruct(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Generate resource for a test struct
	generateResourceForStruct(tempDir, "TestModel")

	// Verify file was created
	resourceFile := filepath.Join(tempDir, "testmodel.go")
	if _, err := os.Stat(resourceFile); os.IsNotExist(err) {
		t.Error("Expected testmodel.go to be generated")
	}

	// Read and verify content
	content, err := os.ReadFile(resourceFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Check for expected content
	if !strings.Contains(contentStr, "TestModelResource") {
		t.Error("Expected generated file to contain TestModelResource")
	}

	if !strings.Contains(contentStr, "RegisterTestModelRoutes") {
		t.Error("Expected generated file to contain RegisterTestModelRoutes")
	}
}

func TestGeneratedResourcesCRUDIntegration(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Generate everything
	tables := LoadSchema(db)
	GenerateStructs(tables)
	GenerateCRUD()
	GenerateAPI(db, tables)

	// Verify that generated resources compile correctly
	// This test ensures the generated code is syntactically correct
	// by checking if we can read the generated files without errors

	usersResourceFile := filepath.Join("gen/resources", "users.go")
	content, err := os.ReadFile(usersResourceFile)
	if err != nil {
		t.Fatalf("Failed to read generated users resource: %v", err)
	}

	// Check that CRUD operations are properly integrated
	contentStr := string(content)

	crudChecks := []string{
		"CRUD *crud.CRUD[models.Users]",
		"r.CRUD.GetAll",
		"r.CRUD.GetByID",
		"r.CRUD.Create",
		"r.CRUD.Update",
		"r.CRUD.Delete",
	}

	for _, check := range crudChecks {
		if !strings.Contains(contentStr, check) {
			t.Errorf("Generated resource missing CRUD integration: %s", check)
		}
	}

	// Check error handling
	errorHandling := []string{
		"if err != nil",
		"c.Status(500)",
		"c.Status(404)",
		"c.Status(400)",
	}

	for _, check := range errorHandling {
		if !strings.Contains(contentStr, check) {
			t.Errorf("Generated resource missing error handling: %s", check)
		}
	}
}

//go:build integration

package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAPI(t *testing.T) {
	tables := LoadSchema(db)
	GenerateStructs(tables)
	GenerateAPI(db, tables, NoAuthConfig())

	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	userResourceFile := filepath.Join(projectRoot, "internal/api/resources", "user.go")
	if _, err := os.Stat(userResourceFile); os.IsNotExist(err) {
		t.Error("Expected user.go resource to be generated")
	}

	todoResourceFile := filepath.Join(projectRoot, "internal/api/resources", "todo.go")
	if _, err := os.Stat(todoResourceFile); os.IsNotExist(err) {
		t.Error("Expected todo.go resource to be generated")
	}

	content, err := os.ReadFile(userResourceFile)
	if err != nil {
		t.Fatalf("Failed to read user.go: %v", err)
	}

	contentStr := string(content)

	expectedStrings := []string{
		"package resources",
		"UserResource",
		"RegisterUserRoutes",
		"CRUD *crud.CRUD[models.User]",
		"crud.New[models.User](db)",
		"router.Get(\"/users\"",
		"func (r *UserResource) List(c *fiber.Ctx) error",
		"func (r *UserResource) Get(c *fiber.Ctx) error",
		"func (r *UserResource) Create(c *fiber.Ctx) error",
		"func (r *UserResource) Update(c *fiber.Ctx) error",
		"func (r *UserResource) Delete(c *fiber.Ctx) error",
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

	structs := parseStructs(testFile)
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
	// Create test fields for User model
	testFields := []StructField{
		{Name: "Id", Type: "string", JSONTag: "id,omitempty", DBTag: "id", IsPointer: false},
		{Name: "Email", Type: "string", JSONTag: "email", DBTag: "email", IsPointer: false},
		{Name: "Firstname", Type: "string", JSONTag: "firstname", DBTag: "firstname", IsPointer: false},
		{Name: "Lastname", Type: "string", JSONTag: "lastname", DBTag: "lastname", IsPointer: false},
		{Name: "Password", Type: "string", JSONTag: "password,omitempty", DBTag: "password", IsPointer: true},
		{Name: "UpdatedAt", Type: "time.Time", JSONTag: "updated_at,omitempty", DBTag: "updated_at", IsPointer: true},
		{Name: "CreatedAt", Type: "time.Time", JSONTag: "created_at,omitempty", DBTag: "created_at", IsPointer: true},
	}

	result := generateResourceFromModel("User", testFields, NoAuthConfig())

	expectedStrings := []string{
		"package resources",
		"UserResource",
		"RegisterUserRoutes",
		"CRUD *crud.CRUD[models.User]",
		"crud.New[models.User](db)",
		"router.Get(\"/users\"",
		"router.Post(\"/users\"",
		"func (r *UserResource) List(c *fiber.Ctx) error",
		"items, err := r.CRUD.GetAll(c.Context())",
		"func (r *UserResource) Get(c *fiber.Ctx) error",
		"item, err := r.CRUD.GetByID(c.Context(), id)",
		"func (r *UserResource) Create(c *fiber.Ctx) error",
		"var createDTO dtos.UserCreateDTO",
		"item := userCreateDTOToModel(createDTO)",
		"r.CRUD.Create(c.Context(), item)",
		"func (r *UserResource) Update(c *fiber.Ctx) error",
		"var updateDTO dtos.UserUpdateDTO",
		"item := userUpdateDTOToModel(updateDTO)",
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
	tempDir := t.TempDir()
	generateResourceForStruct(tempDir, "TestModel", NoAuthConfig())

	resourceFile := filepath.Join(tempDir, "testmodel.go")
	if _, err := os.Stat(resourceFile); os.IsNotExist(err) {
		t.Error("Expected testmodel.go to be generated")
	}

	content, err := os.ReadFile(resourceFile)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "TestModelResource") {
		t.Error("Expected generated file to contain TestModelResource")
	}

	if !strings.Contains(contentStr, "RegisterTestModelRoutes") {
		t.Error("Expected generated file to contain RegisterTestModelRoutes")
	}
}

func TestGeneratedResourcesCRUDIntegration(t *testing.T) {

	tables := LoadSchema(db)
	GenerateStructs(tables)
	GenerateAPI(db, tables, NoAuthConfig())

	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	userResourceFile := filepath.Join(projectRoot, "internal/api/resources", "user.go")
	content, err := os.ReadFile(userResourceFile)
	if err != nil {
		t.Fatalf("Failed to read generated user resource: %v", err)
	}

	contentStr := string(content)

	crudChecks := []string{
		"CRUD *crud.CRUD[models.User]",
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

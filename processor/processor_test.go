package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/sqlite" // Register SQLite driver
	"github.com/nicolasbonnici/gorest/hooks"
	"github.com/nicolasbonnici/gorest/internal/testhelpers"
	"github.com/nicolasbonnici/gorest/query"
)

// Test model
type TestModel struct {
	ID        string    `json:"id" db:"id" rbac:"read:*;write:*"`
	Name      string    `json:"name" db:"name" rbac:"read:*;write:*"`
	Email     string    `json:"email" db:"email" rbac:"read:*;write:*"`
	UserID    *string   `json:"user_id,omitempty" db:"user_id" rbac:"read:*;write:*"`
	CreatedAt time.Time `json:"created_at" db:"created_at" rbac:"read:*;write:none"`
}

func (TestModel) TableName() string { return "test_models" }

// Test DTOs
type TestCreateDTO struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type TestUpdateDTO struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type TestResponseDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	UserID    *string   `json:"user_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Converter functions
func testCreateDTOToModel(dto TestCreateDTO) TestModel {
	// Generate a simple ID for testing
	id := fmt.Sprintf("test-%d", time.Now().UnixNano())
	return TestModel{
		ID:    id,
		Name:  dto.Name,
		Email: dto.Email,
	}
}

func testUpdateDTOToModel(dto TestUpdateDTO) TestModel {
	return TestModel{
		Name:  dto.Name,
		Email: dto.Email,
	}
}

func testModelToDTO(m TestModel) TestResponseDTO {
	return TestResponseDTO(m)
}

// Setup test database
func setupTestDB(t *testing.T) database.Database {
	t.Helper()

	// Create test table schema
	// Note: SQLite doesn't support UUID generation natively, so we rely on the application to set IDs
	schema := `
		CREATE TABLE test_models (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT,
			user_id TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`

	return testhelpers.SetupSQLiteWithSchema(t, schema)
}

// Test configuration creation
func TestNew(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testCRUD := crud.New[TestModel](db)

	// Test with minimal config
	proc := New(ProcessorConfig[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
		DB:   db,
		CRUD: testCRUD,
		Converter: &FuncConverter[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
			CreateToModel: testCreateDTOToModel,
			UpdateToModel: testUpdateDTOToModel,
			ModelToDTO:    testModelToDTO,
		},
	})

	if proc == nil {
		t.Fatal("Expected processor to be created")
	}

	stdProc, ok := proc.(*StandardProcessor[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO])
	if !ok {
		t.Fatal("Expected StandardProcessor type")
	}

	// Check defaults
	if stdProc.config.PaginationLimit != 30 {
		t.Errorf("Expected default PaginationLimit 30, got %d", stdProc.config.PaginationLimit)
	}

	if stdProc.config.PaginationMaxLimit != 100 {
		t.Errorf("Expected default PaginationMaxLimit 100, got %d", stdProc.config.PaginationMaxLimit)
	}

	if stdProc.config.ErrorHandler == nil {
		t.Error("Expected default ErrorHandler to be set")
	}
}

// Test Create operation
func TestCreate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testCRUD := crud.New[TestModel](db)

	proc := New(ProcessorConfig[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
		DB:   db,
		CRUD: testCRUD,
		Converter: &FuncConverter[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
			CreateToModel: testCreateDTOToModel,
			UpdateToModel: testUpdateDTOToModel,
			ModelToDTO:    testModelToDTO,
		},
	})

	app := fiber.New()
	app.Post("/test", proc.Create)

	// Test successful create
	body := `{"name":"Test Name","email":"test@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send test request: %v", err)
	}

	if resp.StatusCode != 201 {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	// Parse response
	bodyBytes, _ := io.ReadAll(resp.Body)
	var result TestResponseDTO
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result.Name != "Test Name" {
		t.Errorf("Expected name 'Test Name', got '%s'", result.Name)
	}
	if result.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", result.Email)
	}
}

// Test Create with validation
func TestCreateWithValidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testCRUD := crud.New[TestModel](db)

	proc := New(ProcessorConfig[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
		DB:   db,
		CRUD: testCRUD,
		Converter: &FuncConverter[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
			CreateToModel: testCreateDTOToModel,
			UpdateToModel: testUpdateDTOToModel,
			ModelToDTO:    testModelToDTO,
		},
		ValidateCreate: func(dto TestCreateDTO) error {
			if dto.Name == "" {
				return fmt.Errorf("name is required")
			}
			return nil
		},
	})

	app := fiber.New()
	app.Post("/test", proc.Create)

	// Test validation failure
	body := `{"email":"test@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send test request: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

// Test Create with custom hook
func TestCreateWithCustomHook(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testCRUD := crud.New[TestModel](db)

	hookCalled := false
	proc := New(ProcessorConfig[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
		DB:   db,
		CRUD: testCRUD,
		Converter: &FuncConverter[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
			CreateToModel: testCreateDTOToModel,
			UpdateToModel: testUpdateDTOToModel,
			ModelToDTO:    testModelToDTO,
		},
	}).WithCreateHook(func(c *fiber.Ctx, dto TestCreateDTO, model *TestModel) error {
		hookCalled = true
		// Modify model in hook
		model.Email = "modified@example.com"
		return nil
	})

	app := fiber.New()
	app.Post("/test", proc.Create)

	body := `{"name":"Test","email":"original@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send test request: %v", err)
	}

	if !hookCalled {
		t.Error("Expected create hook to be called")
	}

	// Verify email was modified by hook
	bodyBytes, _ := io.ReadAll(resp.Body)
	var result TestResponseDTO
	json.Unmarshal(bodyBytes, &result)

	if result.Email != "modified@example.com" {
		t.Errorf("Expected email to be modified by hook to 'modified@example.com', got '%s'", result.Email)
	}
}

// Test invalid request body
func TestCreateInvalidBody(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testCRUD := crud.New[TestModel](db)

	proc := New(ProcessorConfig[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
		DB:   db,
		CRUD: testCRUD,
		Converter: &FuncConverter[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
			CreateToModel: testCreateDTOToModel,
			UpdateToModel: testUpdateDTOToModel,
			ModelToDTO:    testModelToDTO,
		},
	})

	app := fiber.New()
	app.Post("/test", proc.Create)

	// Invalid JSON
	body := `{invalid json}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send test request: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

// Test GetAll with pagination
func TestGetAll(t *testing.T) {
	db := setupTestDB(t)

	// Insert test data
	ctx := context.Background()
	testCRUD := crud.New[TestModel](db)

	for i := 1; i <= 5; i++ {
		model := TestModel{
			ID:    fmt.Sprintf("test-%d", i),
			Name:  fmt.Sprintf("Test %d", i),
			Email: fmt.Sprintf("test%d@example.com", i),
		}
		if err := testCRUD.Create(ctx, model); err != nil {
			t.Fatalf("Failed to create test model %d: %v", i, err)
		}
	}

	proc := New(ProcessorConfig[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
		DB:                 db,
		CRUD:               testCRUD,
		PaginationLimit:    2,
		PaginationMaxLimit: 10,
		AllowedFields:      []string{"name", "email"},
		Converter: &FuncConverter[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
			CreateToModel: testCreateDTOToModel,
			UpdateToModel: testUpdateDTOToModel,
			ModelToDTO:    testModelToDTO,
		},
	})

	app := fiber.New()
	app.Get("/test", proc.GetAll)

	// Test pagination
	req := httptest.NewRequest("GET", "/test?limit=2&page=1", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send test request: %v", err)
	}

	// Parse response
	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d. Response: %s", resp.StatusCode, string(bodyBytes))
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v. Body: %s", err, string(bodyBytes))
	}

	member, ok := result["hydra:member"].([]interface{})
	if !ok {
		t.Fatalf("Expected hydra:member to be an array, got: %T. Full result: %+v", result["hydra:member"], result)
	}

	if len(member) != 2 {
		t.Errorf("Expected 2 items per page, got %d", len(member))
	}
}

// Test GetAll with custom hook
func TestGetAllWithCustomHook(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testCRUD := crud.New[TestModel](db)

	hookCalled := false
	proc := New(ProcessorConfig[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
		DB:   db,
		CRUD: testCRUD,
		Converter: &FuncConverter[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
			CreateToModel: testCreateDTOToModel,
			UpdateToModel: testUpdateDTOToModel,
			ModelToDTO:    testModelToDTO,
		},
	}).WithGetAllHook(func(c *fiber.Ctx, conditions *[]query.Condition, orderBy *[]crud.OrderByClause) error {
		hookCalled = true
		// Add custom condition
		*conditions = append(*conditions, query.Eq("name", "Test"))
		return nil
	})

	app := fiber.New()
	app.Get("/test", proc.GetAll)

	req := httptest.NewRequest("GET", "/test", nil)
	_, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send test request: %v", err)
	}

	if !hookCalled {
		t.Error("Expected GetAll hook to be called")
	}
}

// Test context enrichers
func TestContextEnrichers(t *testing.T) {
	db := setupTestDB(t)

	testCRUD := crud.New[TestModel](db)

	// Custom enricher for testing
	testEnricher := func(c *fiber.Ctx, model interface{}) error {
		testModel := model.(*TestModel)
		userID := "test-user-123"
		testModel.UserID = &userID
		return nil
	}

	proc := New(ProcessorConfig[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
		DB:   db,
		CRUD: testCRUD,
		Converter: &FuncConverter[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
			CreateToModel: testCreateDTOToModel,
			UpdateToModel: testUpdateDTOToModel,
			ModelToDTO:    testModelToDTO,
		},
		ContextEnrichers: []ContextEnricher{testEnricher},
	})

	app := fiber.New()
	app.Post("/test", proc.Create)

	body := `{"name":"Test","email":"test@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send test request: %v", err)
	}

	if resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	bodyBytes, _ := io.ReadAll(resp.Body)
	t.Logf("Response body: %s", string(bodyBytes))
	var result TestResponseDTO
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, body: %s", err, string(bodyBytes))
	}

	t.Logf("Result: %+v", result)
	t.Logf("UserID: %v", result.UserID)

	// Verify the enricher worked by checking the database directly
	ctx := context.Background()
	created, err := testCRUD.GetByID(ctx, result.ID)
	if err != nil {
		t.Fatalf("Failed to fetch created model: %v", err)
	}
	t.Logf("Created model from DB: %+v", created)

	if created.UserID == nil {
		t.Error("Expected user_id to be set by enricher in the database, but it was nil")
	} else if *created.UserID != "test-user-123" {
		t.Errorf("Expected user_id to be 'test-user-123', got '%s'", *created.UserID)
	}
}

// Test FuncConverter
func TestFuncConverter(t *testing.T) {
	converter := &FuncConverter[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
		CreateToModel: testCreateDTOToModel,
		UpdateToModel: testUpdateDTOToModel,
		ModelToDTO:    testModelToDTO,
	}

	// Test CreateDTOToModel
	createDTO := TestCreateDTO{Name: "Test", Email: "test@example.com"}
	model := converter.CreateDTOToModel(createDTO)
	if model.Name != "Test" {
		t.Errorf("Expected name 'Test', got '%s'", model.Name)
	}

	// Test UpdateDTOToModel
	updateDTO := TestUpdateDTO{Name: "Updated", Email: "updated@example.com"}
	model = converter.UpdateDTOToModel(updateDTO)
	if model.Name != "Updated" {
		t.Errorf("Expected name 'Updated', got '%s'", model.Name)
	}

	// Test ModelToResponseDTO
	model = TestModel{ID: "123", Name: "Test", Email: "test@example.com"}
	dto := converter.ModelToResponseDTO(model)
	if dto.ID != "123" {
		t.Errorf("Expected ID '123', got '%s'", dto.ID)
	}

	// Test ModelsToResponseDTOs
	models := []TestModel{
		{ID: "1", Name: "Test 1"},
		{ID: "2", Name: "Test 2"},
	}
	dtos := converter.ModelsToResponseDTOs(models)
	if len(dtos) != 2 {
		t.Errorf("Expected 2 DTOs, got %d", len(dtos))
	}
}

// Test DefaultErrorHandler
func TestDefaultErrorHandler(t *testing.T) {
	handler := &DefaultErrorHandler{}

	// Test parse error
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return handler.HandleError(c, fmt.Errorf("parse error"), "parse")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send test request: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400 for parse error, got %d", resp.StatusCode)
	}
}

// Test hook chaining
func TestHookChaining(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testCRUD := crud.New[TestModel](db)

	proc := New(ProcessorConfig[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
		DB:   db,
		CRUD: testCRUD,
		Converter: &FuncConverter[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
			CreateToModel: testCreateDTOToModel,
			UpdateToModel: testUpdateDTOToModel,
			ModelToDTO:    testModelToDTO,
		},
	}).
		WithCreateHook(func(c *fiber.Ctx, dto TestCreateDTO, model *TestModel) error {
			return nil
		}).
		WithUpdateHook(func(c *fiber.Ctx, dto TestUpdateDTO, model *TestModel) error {
			return nil
		}).
		WithDeleteHook(func(c *fiber.Ctx, id any) error {
			return nil
		}).
		WithGetByIDHook(func(c *fiber.Ctx, id any) error {
			return nil
		}).
		WithGetAllHook(func(c *fiber.Ctx, conditions *[]query.Condition, orderBy *[]crud.OrderByClause) error {
			return nil
		})

	if proc == nil {
		t.Error("Expected processor to be created with chained hooks")
	}

	stdProc, ok := proc.(*StandardProcessor[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO])
	if !ok {
		t.Fatal("Expected StandardProcessor type")
	}

	if stdProc.createHook == nil {
		t.Error("Expected createHook to be set")
	}
	if stdProc.updateHook == nil {
		t.Error("Expected updateHook to be set")
	}
	if stdProc.deleteHook == nil {
		t.Error("Expected deleteHook to be set")
	}
	if stdProc.getByIDHook == nil {
		t.Error("Expected getByIDHook to be set")
	}
	if stdProc.getAllHook == nil {
		t.Error("Expected getAllHook to be set")
	}
}

// Test with hooks layer integration
func TestWithHooksLayer(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Use CRUD with hooks
	testCRUD := crud.NewWithHooks[TestModel](db, hooks.NewNoOpHooks[TestModel]())

	proc := New(ProcessorConfig[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
		DB:   db,
		CRUD: testCRUD,
		Converter: &FuncConverter[TestModel, TestCreateDTO, TestUpdateDTO, TestResponseDTO]{
			CreateToModel: testCreateDTOToModel,
			UpdateToModel: testUpdateDTOToModel,
			ModelToDTO:    testModelToDTO,
		},
	})

	app := fiber.New()
	app.Post("/test", proc.Create)

	body := `{"name":"Test","email":"test@example.com"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to send test request: %v", err)
	}

	if resp.StatusCode != 201 {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
}

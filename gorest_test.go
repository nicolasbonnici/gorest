package gorest

import (
	"context"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/generator"
	"github.com/nicolasbonnici/gorest/logger"
	"github.com/nicolasbonnici/gorest/plugin"
)

type mockLogger struct{}

func (m *mockLogger) Error(msg string, args ...interface{}) {}

type mockDialect struct{}

func (m *mockDialect) Placeholder(n int) string {
	return "?"
}

func (m *mockDialect) SupportsReturning() bool {
	return false
}

func (m *mockDialect) ReturningClause(cols ...string) string {
	return ""
}

func (m *mockDialect) LimitOffset(limit, offset int) string {
	return ""
}

func (m *mockDialect) QuoteIdentifier(name string) string {
	return name
}

func (m *mockDialect) MapType(dbType string) string {
	return dbType
}

func (m *mockDialect) CaseInsensitiveLike() string {
	return "LOWER"
}

type mockRow struct {
	scanFunc func(dest ...interface{}) error
}

func (m *mockRow) Scan(dest ...interface{}) error {
	return m.scanFunc(dest...)
}

type mockRows struct{}

func (m *mockRows) Next() bool                               { return false }
func (m *mockRows) Scan(dest ...interface{}) error           { return nil }
func (m *mockRows) Close() error                             { return nil }
func (m *mockRows) Err() error                               { return nil }

type mockResult struct{}

func (m *mockResult) LastInsertId() (int64, error) { return 0, nil }
func (m *mockResult) RowsAffected() (int64, error) { return 0, nil }

type mockTx struct{}

func (m *mockTx) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	return &mockRows{}, nil
}
func (m *mockTx) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	return &mockRow{}
}
func (m *mockTx) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	return &mockResult{}, nil
}
func (m *mockTx) Commit(ctx context.Context) error   { return nil }
func (m *mockTx) Rollback(ctx context.Context) error { return nil }

type mockDatabase struct {
	pingErr error
}

func (m *mockDatabase) Connect(ctx context.Context, dsn string) error { return nil }
func (m *mockDatabase) Close() error                                  { return nil }
func (m *mockDatabase) Ping(ctx context.Context) error {
	return m.pingErr
}
func (m *mockDatabase) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	return &mockRows{}, nil
}
func (m *mockDatabase) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	return &mockRow{}
}
func (m *mockDatabase) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	return &mockResult{}, nil
}
func (m *mockDatabase) Begin(ctx context.Context) (database.Tx, error) {
	return &mockTx{}, nil
}
func (m *mockDatabase) Dialect() database.Dialect {
	return &mockDialect{}
}
func (m *mockDatabase) DriverName() string {
	return "mock"
}
func (m *mockDatabase) Introspector() database.SchemaIntrospector {
	return nil
}

func TestFindProjectRoot_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gorest-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	root, err := FindProjectRoot()

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if root != tempDir {
		t.Errorf("expected root '%s', got '%s'", tempDir, root)
	}
}

func TestFindProjectRoot_NestedDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gorest-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	nestedDir := filepath.Join(tempDir, "src", "app", "deep")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(nestedDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	root, err := FindProjectRoot()

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if root != tempDir {
		t.Errorf("expected root '%s', got '%s'", tempDir, root)
	}
}

func TestFindProjectRoot_NotFound(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gorest-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	_, err = FindProjectRoot()

	if err == nil {
		t.Error("expected error, got nil")
	}

	if err.Error() != "go.mod not found" {
		t.Errorf("expected 'go.mod not found', got '%s'", err.Error())
	}
}

func TestFindProjectRoot_MultipleGoMod(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "gorest-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	goModPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module parent"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	subDir := filepath.Join(tempDir, "submodule")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	subGoModPath := filepath.Join(subDir, "go.mod")
	if err := os.WriteFile(subGoModPath, []byte("module child"), 0644); err != nil {
		t.Fatalf("failed to create sub go.mod: %v", err)
	}

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	root, err := FindProjectRoot()

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if root != subDir {
		t.Errorf("expected root '%s' (nearest go.mod), got '%s'", subDir, root)
	}
}

func TestConvertColumns_Empty(t *testing.T) {
	dbCols := []database.Column{}
	result := convertColumns(dbCols)

	if len(result) != 0 {
		t.Errorf("expected 0 columns, got %d", len(result))
	}
}

func TestConvertColumns_Single(t *testing.T) {
	dbCols := []database.Column{
		{Name: "id", Type: "integer", IsNullable: false},
	}
	result := convertColumns(dbCols)

	if len(result) != 1 {
		t.Fatalf("expected 1 column, got %d", len(result))
	}

	if result[0].Name != "id" {
		t.Errorf("expected Name 'id', got '%s'", result[0].Name)
	}
	if result[0].Type != "integer" {
		t.Errorf("expected Type 'integer', got '%s'", result[0].Type)
	}
	if result[0].IsNullable != false {
		t.Error("expected IsNullable false")
	}
}

func TestConvertColumns_Multiple(t *testing.T) {
	dbCols := []database.Column{
		{Name: "id", Type: "integer", IsNullable: false},
		{Name: "name", Type: "string", IsNullable: false},
		{Name: "email", Type: "string", IsNullable: true},
		{Name: "created_at", Type: "timestamp", IsNullable: false},
	}
	result := convertColumns(dbCols)

	if len(result) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(result))
	}

	expected := []generator.Column{
		{Name: "id", Type: "integer", IsNullable: false},
		{Name: "name", Type: "string", IsNullable: false},
		{Name: "email", Type: "string", IsNullable: true},
		{Name: "created_at", Type: "timestamp", IsNullable: false},
	}

	for i, exp := range expected {
		if result[i].Name != exp.Name {
			t.Errorf("column %d: expected Name '%s', got '%s'", i, exp.Name, result[i].Name)
		}
		if result[i].Type != exp.Type {
			t.Errorf("column %d: expected Type '%s', got '%s'", i, exp.Type, result[i].Type)
		}
		if result[i].IsNullable != exp.IsNullable {
			t.Errorf("column %d: expected IsNullable %v, got %v", i, exp.IsNullable, result[i].IsNullable)
		}
	}
}

func TestConvertColumns_NullableVariations(t *testing.T) {
	tests := []struct {
		name       string
		isNullable bool
	}{
		{"nullable_column", true},
		{"non_nullable_column", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbCols := []database.Column{
				{Name: tt.name, Type: "string", IsNullable: tt.isNullable},
			}
			result := convertColumns(dbCols)

			if result[0].IsNullable != tt.isNullable {
				t.Errorf("expected IsNullable %v, got %v", tt.isNullable, result[0].IsNullable)
			}
		})
	}
}

func TestConvertRelations_Empty(t *testing.T) {
	dbRels := []database.Relation{}
	result := convertRelations(dbRels)

	if len(result) != 0 {
		t.Errorf("expected 0 relations, got %d", len(result))
	}
}

func TestConvertRelations_Single(t *testing.T) {
	dbRels := []database.Relation{
		{
			ChildTable:   "orders",
			ChildColumn:  "user_id",
			ParentTable:  "users",
			ParentColumn: "id",
		},
	}
	result := convertRelations(dbRels)

	if len(result) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(result))
	}

	if result[0].ChildTable != "orders" {
		t.Errorf("expected ChildTable 'orders', got '%s'", result[0].ChildTable)
	}
	if result[0].ChildColumn != "user_id" {
		t.Errorf("expected ChildColumn 'user_id', got '%s'", result[0].ChildColumn)
	}
	if result[0].ParentTable != "users" {
		t.Errorf("expected ParentTable 'users', got '%s'", result[0].ParentTable)
	}
	if result[0].ParentColumn != "id" {
		t.Errorf("expected ParentColumn 'id', got '%s'", result[0].ParentColumn)
	}
}

func TestConvertRelations_Multiple(t *testing.T) {
	dbRels := []database.Relation{
		{
			ChildTable:   "orders",
			ChildColumn:  "user_id",
			ParentTable:  "users",
			ParentColumn: "id",
		},
		{
			ChildTable:   "order_items",
			ChildColumn:  "order_id",
			ParentTable:  "orders",
			ParentColumn: "id",
		},
		{
			ChildTable:   "order_items",
			ChildColumn:  "product_id",
			ParentTable:  "products",
			ParentColumn: "id",
		},
	}
	result := convertRelations(dbRels)

	if len(result) != 3 {
		t.Fatalf("expected 3 relations, got %d", len(result))
	}

	expected := []generator.Relation{
		{ChildTable: "orders", ChildColumn: "user_id", ParentTable: "users", ParentColumn: "id"},
		{ChildTable: "order_items", ChildColumn: "order_id", ParentTable: "orders", ParentColumn: "id"},
		{ChildTable: "order_items", ChildColumn: "product_id", ParentTable: "products", ParentColumn: "id"},
	}

	for i, exp := range expected {
		if result[i].ChildTable != exp.ChildTable {
			t.Errorf("relation %d: expected ChildTable '%s', got '%s'", i, exp.ChildTable, result[i].ChildTable)
		}
		if result[i].ChildColumn != exp.ChildColumn {
			t.Errorf("relation %d: expected ChildColumn '%s', got '%s'", i, exp.ChildColumn, result[i].ChildColumn)
		}
		if result[i].ParentTable != exp.ParentTable {
			t.Errorf("relation %d: expected ParentTable '%s', got '%s'", i, exp.ParentTable, result[i].ParentTable)
		}
		if result[i].ParentColumn != exp.ParentColumn {
			t.Errorf("relation %d: expected ParentColumn '%s', got '%s'", i, exp.ParentColumn, result[i].ParentColumn)
		}
	}
}

func TestConvertRelations_SelfReferential(t *testing.T) {
	dbRels := []database.Relation{
		{
			ChildTable:   "categories",
			ChildColumn:  "parent_id",
			ParentTable:  "categories",
			ParentColumn: "id",
		},
	}
	result := convertRelations(dbRels)

	if len(result) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(result))
	}

	if result[0].ChildTable != result[0].ParentTable {
		t.Error("expected self-referential relation")
	}
}

func TestSetupHealthCheck_Healthy(t *testing.T) {
	app := fiber.New()
	db := &mockDatabase{pingErr: nil}
	log := &mockLogger{}

	SetupHealthCheck(app, db, log)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !contains(bodyStr, "healthy") {
		t.Error("expected 'healthy' in response body")
	}

	if !contains(bodyStr, "up") {
		t.Error("expected database status 'up' in response body")
	}
}

func TestSetupHealthCheck_Unhealthy(t *testing.T) {
	app := fiber.New()
	db := &mockDatabase{pingErr: fiber.NewError(503, "database unavailable")}
	log := &mockLogger{}

	SetupHealthCheck(app, db, log)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.StatusCode != 503 {
		t.Errorf("expected status 503, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !contains(bodyStr, "unhealthy") {
		t.Error("expected 'unhealthy' in response body")
	}

	if !contains(bodyStr, "down") {
		t.Error("expected database status 'down' in response body")
	}
}

func TestSetupOpenAPIUI_Success(t *testing.T) {
	app := fiber.New()

	SetupOpenAPIUI(app)

	req := httptest.NewRequest("GET", "/openapi", nil)
	resp, err := app.Test(req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/html" {
		t.Errorf("expected Content-Type 'text/html', got '%s'", contentType)
	}

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	if !contains(bodyStr, "<!DOCTYPE html>") {
		t.Error("expected HTML doctype in response")
	}

	if !contains(bodyStr, "GoREST API Documentation") {
		t.Error("expected page title in response")
	}

	if !contains(bodyStr, "/openapi.json") {
		t.Error("expected OpenAPI JSON URL in response")
	}

	if !contains(bodyStr, "@scalar/api-reference") {
		t.Error("expected Scalar API reference script in response")
	}
}

func TestSetupOpenAPIUI_ResponseStructure(t *testing.T) {
	app := fiber.New()

	SetupOpenAPIUI(app)

	req := httptest.NewRequest("GET", "/openapi", nil)
	resp, _ := app.Test(req)

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	requiredElements := []string{
		"<html>",
		"<head>",
		"<title>",
		"<body>",
		"<script",
		"data-url",
	}

	for _, elem := range requiredElements {
		if !contains(bodyStr, elem) {
			t.Errorf("expected '%s' in HTML response", elem)
		}
	}
}

func TestVersion_Default(t *testing.T) {
	if Version != "dev" {
		t.Errorf("expected default Version 'dev', got '%s'", Version)
	}
}

func TestConfig_DefaultConfigPath(t *testing.T) {
	cfg := Config{}

	if cfg.ConfigPath != "" {
		t.Errorf("expected empty ConfigPath by default, got '%s'", cfg.ConfigPath)
	}
}

func TestConfig_WithConfigPath(t *testing.T) {
	cfg := Config{
		ConfigPath: "/custom/path",
	}

	if cfg.ConfigPath != "/custom/path" {
		t.Errorf("expected ConfigPath '/custom/path', got '%s'", cfg.ConfigPath)
	}
}

func TestConfig_WithRegisterRoutes(t *testing.T) {
	called := false
	cfg := Config{
		RegisterRoutes: func(app *fiber.App, db database.Database, paginationLimit, paginationMaxLimit int, pluginRegistry *plugin.PluginRegistry) {
			called = true
		},
	}

	if cfg.RegisterRoutes == nil {
		t.Fatal("expected RegisterRoutes to be set")
	}

	cfg.RegisterRoutes(nil, nil, 0, 0, nil)

	if !called {
		t.Error("expected RegisterRoutes function to be called")
	}
}

func TestSetupHealthCheck_ErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		pingErr        error
		expectedStatus int
	}{
		{"healthy database", nil, 200},
		{"unhealthy database", fiber.NewError(500, "db error"), 503},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			db := &mockDatabase{pingErr: tt.pingErr}
			log := &mockLogger{}

			SetupHealthCheck(app, db, log)

			req := httptest.NewRequest("GET", "/health", nil)
			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestConvertColumns_TypePreservation(t *testing.T) {
	types := []string{"integer", "string", "boolean", "timestamp", "uuid", "json"}

	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			dbCols := []database.Column{
				{Name: "test_col", Type: typ, IsNullable: false},
			}
			result := convertColumns(dbCols)

			if result[0].Type != typ {
				t.Errorf("expected Type '%s', got '%s'", typ, result[0].Type)
			}
		})
	}
}

func TestConvertRelations_ComplexHierarchy(t *testing.T) {
	dbRels := []database.Relation{
		{ChildTable: "users", ChildColumn: "company_id", ParentTable: "companies", ParentColumn: "id"},
		{ChildTable: "orders", ChildColumn: "user_id", ParentTable: "users", ParentColumn: "id"},
		{ChildTable: "order_items", ChildColumn: "order_id", ParentTable: "orders", ParentColumn: "id"},
		{ChildTable: "order_items", ChildColumn: "product_id", ParentTable: "products", ParentColumn: "id"},
		{ChildTable: "products", ChildColumn: "category_id", ParentTable: "categories", ParentColumn: "id"},
	}

	result := convertRelations(dbRels)

	if len(result) != 5 {
		t.Errorf("expected 5 relations, got %d", len(result))
	}

	for i, rel := range result {
		if rel.ChildTable == "" || rel.ParentTable == "" {
			t.Errorf("relation %d has empty table name", i)
		}
		if rel.ChildColumn == "" || rel.ParentColumn == "" {
			t.Errorf("relation %d has empty column name", i)
		}
	}
}

func TestSetupHealthCheck_ContextTimeout(t *testing.T) {
	app := fiber.New()
	db := &mockDatabase{pingErr: nil}

	SetupHealthCheck(app, db, logger.Log)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req, 5000)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestSetupOpenAPIUI_MultipleRequests(t *testing.T) {
	app := fiber.New()
	SetupOpenAPIUI(app)

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/openapi", nil)
		resp, err := app.Test(req)

		if err != nil {
			t.Fatalf("request %d: expected no error, got %v", i, err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("request %d: expected status 200, got %d", i, resp.StatusCode)
		}
	}
}

func TestConvertColumns_LargeDataset(t *testing.T) {
	dbCols := make([]database.Column, 100)
	for i := 0; i < 100; i++ {
		dbCols[i] = database.Column{
			Name:       "col_" + string(rune('0'+i)),
			Type:       "string",
			IsNullable: i%2 == 0,
		}
	}

	result := convertColumns(dbCols)

	if len(result) != 100 {
		t.Errorf("expected 100 columns, got %d", len(result))
	}

	for i := 0; i < 100; i++ {
		if result[i].IsNullable != (i%2 == 0) {
			t.Errorf("column %d: IsNullable mismatch", i)
		}
	}
}

func TestConvertRelations_DuplicateRelations(t *testing.T) {
	dbRels := []database.Relation{
		{ChildTable: "orders", ChildColumn: "user_id", ParentTable: "users", ParentColumn: "id"},
		{ChildTable: "orders", ChildColumn: "user_id", ParentTable: "users", ParentColumn: "id"},
	}

	result := convertRelations(dbRels)

	if len(result) != 2 {
		t.Errorf("expected 2 relations (including duplicate), got %d", len(result))
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

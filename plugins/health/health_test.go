package health

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/database"
)

// mockDatabase implements a minimal database.Database interface for testing
type mockDatabase struct {
	pingError error
}

func (m *mockDatabase) Ping(ctx context.Context) error {
	return m.pingError
}

func (m *mockDatabase) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	return nil
}

func (m *mockDatabase) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	return nil, nil
}

func (m *mockDatabase) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	return nil, nil
}

func (m *mockDatabase) Close() error {
	return nil
}

func (m *mockDatabase) Dialect() database.Dialect {
	return nil
}

func (m *mockDatabase) Begin(ctx context.Context) (database.Tx, error) {
	return nil, nil
}

func (m *mockDatabase) Connect(ctx context.Context, connStr string) error {
	return nil
}

func (m *mockDatabase) DriverName() string {
	return "mock"
}

type mockIntrospector struct{}

func (m *mockIntrospector) LoadSchema(ctx context.Context) ([]database.TableSchema, error) {
	return nil, nil
}

func (m *mockIntrospector) GetColumns(ctx context.Context, tableName string) ([]database.Column, error) {
	return nil, nil
}

func (m *mockIntrospector) GetRelations(ctx context.Context) ([]database.Relation, error) {
	return nil, nil
}

func (m *mockDatabase) Introspector() database.SchemaIntrospector {
	return &mockIntrospector{}
}

func TestHealthPlugin_Name(t *testing.T) {
	plugin := NewPlugin()
	if name := plugin.Name(); name != "health" {
		t.Errorf("expected plugin name 'health', got '%s'", name)
	}
}

func TestHealthPlugin_Initialize(t *testing.T) {
	plugin := NewPlugin().(*HealthPlugin)

	config := map[string]interface{}{
		"database":   &mockDatabase{},
		"__version": "1.0.0",
	}

	err := plugin.Initialize(config)
	if err != nil {
		t.Errorf("Initialize failed: %v", err)
	}

	// Check version was set
	if plugin.version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", plugin.version)
	}
}

func TestHealthPlugin_HealthCheckWithDatabase(t *testing.T) {
	app := fiber.New()
	plugin := NewPlugin().(*HealthPlugin)

	// Initialize with healthy database
	config := map[string]interface{}{
		"database":   &mockDatabase{pingError: nil},
		"__version": "test",
	}
	plugin.Initialize(config)
	plugin.SetupEndpoints(app)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Check security headers
	headers := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-Xss-Protection",
		"Strict-Transport-Security",
		"Content-Security-Policy",
		"Referrer-Policy",
		"Permissions-Policy",
	}

	for _, header := range headers {
		if value := resp.Header.Get(header); value == "" {
			t.Errorf("security header '%s' not set", header)
		}
	}
}

func TestHealthPlugin_HealthCheckDatabaseDown(t *testing.T) {
	app := fiber.New()
	plugin := &HealthPlugin{
		db:      &mockDatabase{pingError: errors.New("connection failed")},
		version: "test",
	}
	plugin.SetupEndpoints(app)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 503 {
		t.Errorf("expected status 503, got %d", resp.StatusCode)
	}
}

func TestHealthPlugin_HealthCheckNoDatabase(t *testing.T) {
	app := fiber.New()
	plugin := NewPlugin().(*HealthPlugin)

	// Initialize without database
	config := map[string]interface{}{
		"__version": "test",
	}
	plugin.Initialize(config)
	plugin.SetupEndpoints(app)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestHealthPlugin_TraceMethodBlocked(t *testing.T) {
	app := fiber.New()
	plugin := NewPlugin().(*HealthPlugin)

	config := map[string]interface{}{
		"__version": "test",
	}
	plugin.Initialize(config)
	plugin.SetupEndpoints(app)

	req := httptest.NewRequest("TRACE", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != 405 {
		t.Errorf("expected status 405 for TRACE method, got %d", resp.StatusCode)
	}
}

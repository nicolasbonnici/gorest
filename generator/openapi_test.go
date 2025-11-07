//go:build integration

package generator

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestSetupOpenAPI(t *testing.T) {
	app := fiber.New()

	tables := map[string]TableSchema{}
	SetupOpenAPI(app, tables, 100, 1000)

	// Test that endpoint is registered
	req := httptest.NewRequest("GET", "/openapi.json", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test request: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify response is valid JSON
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify OpenAPI structure exists
	if result["openapi"] == nil {
		t.Error("Expected 'openapi' field in response")
	}

	if result["paths"] == nil {
		t.Error("Expected 'paths' field in response")
	}

	if result["components"] == nil {
		t.Error("Expected 'components' field in response")
	}
}

func TestBuildSchemaPropertiesFromDTO(t *testing.T) {
	fields := []StructField{
		{
			Name:      "ID",
			Type:      "string",
			JSONTag:   "id",
			IsPointer: false,
		},
		{
			Name:      "Title",
			Type:      "string",
			JSONTag:   "title",
			IsPointer: false,
		},
		{
			Name:      "Description",
			Type:      "string",
			JSONTag:   "description",
			IsPointer: true,
		},
		{
			Name:      "Count",
			Type:      "int",
			JSONTag:   "count",
			IsPointer: false,
		},
	}

	properties := buildSchemaPropertiesFromDTO(fields)

	if len(properties) != 4 {
		t.Errorf("Expected 4 properties, got %d", len(properties))
	}

	// Verify id property
	idProp, ok := properties["id"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected id property to be a map")
	}
	if idProp["type"] != "string" {
		t.Errorf("Expected id type to be string, got %v", idProp["type"])
	}
	if idProp["nullable"] != false {
		t.Error("Expected id nullable to be false")
	}

	// Verify nullable field
	descProp, ok := properties["description"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected description property to be a map")
	}
	if descProp["nullable"] != true {
		t.Error("Expected description nullable to be true")
	}

	// Verify integer type
	countProp, ok := properties["count"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected count property to be a map")
	}
	if countProp["type"] != "integer" {
		t.Errorf("Expected count type to be integer, got %v", countProp["type"])
	}
}

func TestGetRequiredFieldsFromDTO(t *testing.T) {
	fields := []StructField{
		{Name: "ID", Type: "string", JSONTag: "id", IsPointer: false},
		{Name: "Title", Type: "string", JSONTag: "title", IsPointer: false},
		{Name: "Description", Type: "string", JSONTag: "description", IsPointer: true},
		{Name: "CreatedAt", Type: "time.Time", JSONTag: "created_at", IsPointer: false},
		{Name: "UpdatedAt", Type: "time.Time", JSONTag: "updated_at", IsPointer: true},
		{Name: "Count", Type: "int", JSONTag: "count", IsPointer: false},
	}

	required := getRequiredFieldsFromDTO(fields)

	// Should include: title, count
	// Should exclude: id, created_at, updated_at, description (pointer)
	if len(required) != 2 {
		t.Errorf("Expected 2 required fields, got %d: %v", len(required), required)
	}

	// Verify title is required
	titleFound := false
	countFound := false
	for _, field := range required {
		if field == "title" {
			titleFound = true
		}
		if field == "count" {
			countFound = true
		}
	}

	if !titleFound {
		t.Error("Expected 'title' to be in required fields")
	}
	if !countFound {
		t.Error("Expected 'count' to be in required fields")
	}

	// Verify id, created_at, updated_at are not required
	for _, field := range required {
		if field == "id" || field == "created_at" || field == "updated_at" {
			t.Errorf("Field '%s' should not be in required fields", field)
		}
	}
}

func TestGoTypeToOpenAPIType(t *testing.T) {
	tests := []struct {
		name           string
		goType         string
		expectedType   string
		expectedFormat string
	}{
		{"string type", "string", "string", ""},
		{"int type", "int", "integer", "int32"},
		{"int64 type", "int64", "integer", "int64"},
		{"float64 type", "float64", "number", "double"},
		{"bool type", "bool", "boolean", ""},
		{"time.Time type", "time.Time", "string", "date-time"},
		{"unknown type", "CustomType", "string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ, format := GoTypeToOpenAPIType(tt.goType)

			if typ != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, typ)
			}

			if format != tt.expectedFormat {
				t.Errorf("Expected format %s, got %s", tt.expectedFormat, format)
			}
		})
	}
}

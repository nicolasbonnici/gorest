package formatter

import (
	"encoding/json"
	"testing"
)

type TestModel struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func TestJSONFormatter(t *testing.T) {
	formatter := &JSONFormatter{}

	data := TestModel{
		ID:    "1",
		Title: "Test Item",
	}

	result, err := formatter.Format(data, "/testmodels")
	if err != nil {
		t.Fatalf("JSON formatting failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if decoded["id"] != "1" {
		t.Errorf("Expected id=1, got %v", decoded["id"])
	}

	if decoded["title"] != "Test Item" {
		t.Errorf("Expected title='Test Item', got %v", decoded["title"])
	}

	if formatter.ContentType() != "application/json" {
		t.Errorf("Expected content type application/json, got %s", formatter.ContentType())
	}
}

func TestJSONLDFormatterSingleItem(t *testing.T) {
	formatter := &JSONLDFormatter{}

	data := TestModel{
		ID:    "1",
		Title: "Test Item",
	}

	result, err := formatter.Format(data, "/testmodels")
	if err != nil {
		t.Fatalf("JSON-LD formatting failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatalf("Failed to decode JSON-LD: %v", err)
	}

	// Check @context
	if decoded["@context"] != "https://schema.org/" {
		t.Errorf("Expected @context='https://schema.org/', got %v", decoded["@context"])
	}

	// Check @type is set to the struct name
	if decoded["@type"] != "TestModel" {
		t.Errorf("Expected @type='TestModel', got %v", decoded["@type"])
	}

	// Check @id is in format /resource/uuid
	expectedIRI := "/testmodels/1"
	if decoded["@id"] != expectedIRI {
		t.Errorf("Expected @id='%s', got %v", expectedIRI, decoded["@id"])
	}

	// Check data fields
	if decoded["id"] != "1" {
		t.Errorf("Expected id=1, got %v", decoded["id"])
	}

	if decoded["title"] != "Test Item" {
		t.Errorf("Expected title='Test Item', got %v", decoded["title"])
	}

	if formatter.ContentType() != "application/ld+json" {
		t.Errorf("Expected content type application/ld+json, got %s", formatter.ContentType())
	}
}

func TestJSONLDFormatterCollection(t *testing.T) {
	formatter := &JSONLDFormatter{}

	data := []TestModel{
		{ID: "1", Title: "First"},
		{ID: "2", Title: "Second"},
	}

	result, err := formatter.Format(data, "/testmodels")
	if err != nil {
		t.Fatalf("JSON-LD formatting failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatalf("Failed to decode JSON-LD: %v", err)
	}

	// Check @context
	if decoded["@context"] != "https://schema.org/" {
		t.Errorf("Expected @context='https://schema.org/', got %v", decoded["@context"])
	}

	// Check @graph for collections
	graph, ok := decoded["@graph"]
	if !ok {
		t.Fatal("Expected @graph field in JSON-LD collection output")
	}

	items, ok := graph.([]interface{})
	if !ok {
		t.Fatal("Expected @graph to be an array")
	}

	if len(items) != 2 {
		t.Fatalf("Expected 2 items in @graph, got %d", len(items))
	}

	// Check first item
	firstItem := items[0].(map[string]interface{})
	if firstItem["id"] != "1" {
		t.Errorf("Expected first item id=1, got %v", firstItem["id"])
	}
	if firstItem["@type"] != "TestModel" {
		t.Errorf("Expected @type='TestModel', got %v", firstItem["@type"])
	}
	expectedIRI := "/testmodels/1"
	if firstItem["@id"] != expectedIRI {
		t.Errorf("Expected first item @id='%s', got %v", expectedIRI, firstItem["@id"])
	}
}

func TestGetFormatter(t *testing.T) {
	tests := []struct {
		format           string
		expectedJSON     bool
		expectedContentType string
	}{
		{"json", true, "application/json"},
		{"jsonld", false, "application/ld+json"},
		{"json-ld", false, "application/ld+json"},
		{"", false, "application/ld+json"}, // default
		{"unknown", false, "application/ld+json"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			formatter := GetFormatter(tt.format)
			if formatter == nil {
				t.Fatal("Formatter should not be nil")
			}

			contentType := formatter.ContentType()
			if contentType != tt.expectedContentType {
				t.Errorf("Expected content type %s, got %s", tt.expectedContentType, contentType)
			}

			// Check type by content type
			isJSON := (contentType == "application/json")
			if isJSON != tt.expectedJSON {
				t.Errorf("Expected isJSON=%v, got %v", tt.expectedJSON, isJSON)
			}
		})
	}
}

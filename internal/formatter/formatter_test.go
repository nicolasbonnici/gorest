package formatter

import (
	"encoding/json"
	"strings"
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

	if decoded["@context"] != "https://schema.org/" {
		t.Errorf("Expected @context='https://schema.org/', got %v", decoded["@context"])
	}

	if decoded["@type"] != "TestModel" {
		t.Errorf("Expected @type='TestModel', got %v", decoded["@type"])
	}

	expectedIRI := "/testmodels/1"
	if decoded["@id"] != expectedIRI {
		t.Errorf("Expected @id='%s', got %v", expectedIRI, decoded["@id"])
	}

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

	if decoded["@context"] != "https://schema.org/" {
		t.Errorf("Expected @context='https://schema.org/', got %v", decoded["@context"])
	}

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

func TestJSONLDFormatterWithIDInPath(t *testing.T) {
	formatter := &JSONLDFormatter{}

	data := TestModel{
		ID:    "0199da00-8bde-7611-a4a6-1e3df90e95ce",
		Title: "Test Item",
	}

	// Path already contains the ID (like when getting a single item by ID)
	result, err := formatter.Format(data, "/todos/0199da00-8bde-7611-a4a6-1e3df90e95ce")
	if err != nil {
		t.Fatalf("JSON-LD formatting failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatalf("Failed to decode JSON-LD: %v", err)
	}

	expectedIRI := "/todos/0199da00-8bde-7611-a4a6-1e3df90e95ce"
	if decoded["@id"] != expectedIRI {
		t.Errorf("Expected @id='%s', got %v", expectedIRI, decoded["@id"])
	}

	if strings.Contains(decoded["@id"].(string), "0199da00-8bde-7611-a4a6-1e3df90e95ce/0199da00-8bde-7611-a4a6-1e3df90e95ce") {
		t.Errorf("IRI contains duplicate ID: %v", decoded["@id"])
	}
}

func TestJSONLDFormatterWithTrailingSlash(t *testing.T) {
	formatter := &JSONLDFormatter{}

	data := TestModel{
		ID:    "0199da00-8bde-7611-a4a6-1e3df90e95ce",
		Title: "Test Item",
	}

	// Path with trailing slash (common issue)
	result, err := formatter.Format(data, "/todos/")
	if err != nil {
		t.Fatalf("JSON-LD formatting failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatalf("Failed to decode JSON-LD: %v", err)
	}

	expectedIRI := "/todos/0199da00-8bde-7611-a4a6-1e3df90e95ce"
	if decoded["@id"] != expectedIRI {
		t.Errorf("Expected @id='%s', got %v", expectedIRI, decoded["@id"])
	}

	if strings.Contains(decoded["@id"].(string), "//") {
		t.Errorf("IRI contains double slashes: %v", decoded["@id"])
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

			isJSON := (contentType == "application/json")
			if isJSON != tt.expectedJSON {
				t.Errorf("Expected isJSON=%v, got %v", tt.expectedJSON, isJSON)
			}
		})
	}
}

func TestJSONLDFormatterForeignKeyIRI(t *testing.T) {
	type Todo struct {
		ID      string `json:"id"`
		UserID  string `json:"userId"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	formatter := &JSONLDFormatter{}
	todo := Todo{
		ID:      "todo-123",
		UserID:  "user-456",
		Title:   "Test Todo",
		Content: "Test Content",
	}

	output, err := formatter.Format(todo, "/todos")
	if err != nil {
		t.Fatalf("Failed to format: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	userID, ok := result["userId"].(string)
	if !ok {
		t.Fatal("userId should be a string")
	}

	expectedUserID := "/users/user-456"
	if userID != expectedUserID {
		t.Errorf("Expected userId to be IRI %s, got %s", expectedUserID, userID)
	}

	if !strings.HasPrefix(userID, "/users/") {
		t.Error("userId should be converted to IRI starting with /users/")
	}
}

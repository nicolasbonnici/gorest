package serializer

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
	"github.com/nicolasbonnici/gorest/internal/testhelpers"
)

type TestModel struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func TestJSONSerializer(t *testing.T) {
	serializer := &JSONSerializer{}

	data := TestModel{
		ID:    "1",
		Title: "Test Item",
	}

	result, err := serializer.Serialize(data, "/testmodels")
	if err != nil {
		t.Fatalf("JSON serialization failed: %v", err)
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

	if serializer.ContentType() != "application/json" {
		t.Errorf("Expected content type application/json, got %s", serializer.ContentType())
	}
}

func TestJSONLDSerializerSingleItem(t *testing.T) {
	serializer := &JSONLDSerializer{}

	data := TestModel{
		ID:    "1",
		Title: "Test Item",
	}

	result, err := serializer.Serialize(data, "/testmodels")
	if err != nil {
		t.Fatalf("JSON-LD serialization failed: %v", err)
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

	if serializer.ContentType() != "application/ld+json" {
		t.Errorf("Expected content type application/ld+json, got %s", serializer.ContentType())
	}
}

func TestJSONLDSerializerCollection(t *testing.T) {
	serializer := &JSONLDSerializer{}

	data := []TestModel{
		{ID: "1", Title: "First"},
		{ID: "2", Title: "Second"},
	}

	result, err := serializer.Serialize(data, "/testmodels")
	if err != nil {
		t.Fatalf("JSON-LD serialization failed: %v", err)
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

func TestJSONLDSerializerWithIDInPath(t *testing.T) {
	serializer := &JSONLDSerializer{}

	data := TestModel{
		ID:    "0199da00-8bde-7611-a4a6-1e3df90e95ce",
		Title: "Test Item",
	}

	// Path already contains the ID (like when getting a single item by ID)
	result, err := serializer.Serialize(data, "/todos/0199da00-8bde-7611-a4a6-1e3df90e95ce")
	if err != nil {
		t.Fatalf("JSON-LD serialization failed: %v", err)
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

func TestJSONLDSerializerWithTrailingSlash(t *testing.T) {
	serializer := &JSONLDSerializer{}

	data := TestModel{
		ID:    "0199da00-8bde-7611-a4a6-1e3df90e95ce",
		Title: "Test Item",
	}

	// Path with trailing slash (common issue)
	result, err := serializer.Serialize(data, "/todos/")
	if err != nil {
		t.Fatalf("JSON-LD serialization failed: %v", err)
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

func TestGetSerializer(t *testing.T) {
	tests := []struct {
		format              string
		expectedJSON        bool
		expectedContentType string
	}{
		{"json", true, "application/json"},
		{"jsonld", false, "application/ld+json"},
		{"json-ld", false, "application/ld+json"},
		{"", false, "application/ld+json"},        // default
		{"unknown", false, "application/ld+json"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			serializer := GetSerializer(tt.format)
			if serializer == nil {
				t.Fatal("Serializer should not be nil")
			}

			contentType := serializer.ContentType()
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

func TestJSONLDSerializerForeignKeyIRI(t *testing.T) {
	type Todo struct {
		ID      string `json:"id"`
		UserID  string `json:"userId"`
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	serializer := &JSONLDSerializer{}
	todo := Todo{
		ID:      "todo-123",
		UserID:  "user-456",
		Title:   "Test Todo",
		Content: "Test Content",
	}

	output, err := serializer.Serialize(todo, "/todos")
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if _, exists := result["userId"]; exists {
		t.Error("userId field should be removed and replaced with user")
	}

	user, ok := result["user"].(string)
	if !ok {
		t.Fatal("user should be a string IRI")
	}

	expectedUserIRI := "/users/user-456"
	if user != expectedUserIRI {
		t.Errorf("Expected user to be IRI %s, got %s", expectedUserIRI, user)
	}

	if !strings.HasPrefix(user, "/users/") {
		t.Error("user should be converted to IRI starting with /users/")
	}
}

func TestParseExpand(t *testing.T) {
	configs := map[string]RelationConfig{
		"user": {
			Field:           "user",
			ForeignKeyField: "userId",
		},
		"comments": {
			Field:           "comments",
			ForeignKeyField: "postId",
		},
	}

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "valid relations",
			input:    []string{"user", "comments"},
			expected: []string{"user", "comments"},
		},
		{
			name:     "invalid relation filtered out",
			input:    []string{"user", "invalid"},
			expected: []string{"user"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseExpand(tt.input, configs)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d results, got %d", len(tt.expected), len(result))
			}

			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("expected %s at index %d, got %s", exp, i, result[i])
				}
			}
		})
	}
}

type User struct {
	ID   string `json:"id" db:"id" rbac:"read:*;write:*"`
	Name string `json:"name" db:"name" rbac:"read:*;write:*"`
}

func (User) TableName() string { return "users" }

type Post struct {
	ID      string  `json:"id" db:"id" rbac:"read:*;write:*"`
	UserID  *string `json:"userId" db:"user_id" rbac:"read:*;write:*"`
	Title   string  `json:"title" db:"title" rbac:"read:*;write:*"`
	Content string  `json:"content" db:"content" rbac:"read:*;write:*"`
}

func (Post) TableName() string { return "posts" }

func setupTestDB(t *testing.T) (database.Database, func()) {
	schema := `
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL
		);
		CREATE TABLE posts (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			title TEXT NOT NULL,
			content TEXT NOT NULL
		);
	`

	db := testhelpers.SetupSQLiteWithSchema(t, schema)
	ctx := context.Background()

	_, err := db.Exec(ctx, `INSERT INTO users (id, name) VALUES ('user-1', 'Alice')`)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}

	userID := "user-1"
	_, err = db.Exec(ctx, `INSERT INTO posts (id, user_id, title, content) VALUES ('post-1', ?, 'Test Post', 'Content')`, userID)
	if err != nil {
		t.Fatalf("failed to insert test post: %v", err)
	}

	cleanup := func() {
		testhelpers.Cleanup(t, db)
	}

	return db, cleanup
}

func TestExpandRelations_Single(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	postCRUD := crud.New[Post](db)
	userCRUD := crud.New[User](db)

	ctx := context.Background()
	post, err := postCRUD.GetByID(ctx, "post-1")
	if err != nil {
		t.Fatalf("failed to get post: %v", err)
	}

	configs := map[string]RelationConfig{
		"user": {
			Field:           "user",
			ForeignKeyField: "userId",
			RelatedTable:    "users",
			Fetcher:         crud.RelationFetcher(userCRUD),
		},
	}

	expanded, err := ExpandRelations(ctx, *post, []string{"user"}, configs)
	if err != nil {
		t.Fatalf("failed to expand: %v", err)
	}

	expandedMap, ok := expanded.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}

	if _, hasUserID := expandedMap["userId"]; hasUserID {
		t.Error("userId field should be removed when user is expanded")
	}

	userField, ok := expandedMap["user"]
	if !ok {
		t.Fatal("user field not found in expanded result")
	}

	userBytes, _ := json.Marshal(userField)
	var user User
	json.Unmarshal(userBytes, &user)

	if user.ID != "user-1" {
		t.Errorf("expected user ID user-1, got %s", user.ID)
	}

	if user.Name != "Alice" {
		t.Errorf("expected user name Alice, got %s", user.Name)
	}
}

func TestExpandRelations_Slice(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	postCRUD := crud.New[Post](db)
	userCRUD := crud.New[User](db)

	ctx := context.Background()
	result, err := postCRUD.GetAllPaginated(ctx, crud.PaginationOptions{
		Limit:        10,
		Offset:       0,
		IncludeCount: false,
	})
	if err != nil {
		t.Fatalf("failed to get posts: %v", err)
	}

	configs := map[string]RelationConfig{
		"user": {
			Field:           "user",
			ForeignKeyField: "userId",
			RelatedTable:    "users",
			Fetcher:         crud.RelationFetcher(userCRUD),
		},
	}

	expanded, err := ExpandRelations(ctx, result.Items, []string{"user"}, configs)
	if err != nil {
		t.Fatalf("failed to expand: %v", err)
	}

	expandedSlice, ok := expanded.([]interface{})
	if !ok {
		t.Fatal("expected slice result")
	}

	if len(expandedSlice) != 1 {
		t.Fatalf("expected 1 post, got %d", len(expandedSlice))
	}

	firstPost := expandedSlice[0].(map[string]interface{})

	if _, hasUserID := firstPost["userId"]; hasUserID {
		t.Error("userId field should be removed when user is expanded")
	}

	userField, ok := firstPost["user"]
	if !ok {
		t.Fatal("user field not found in first post")
	}

	userBytes, _ := json.Marshal(userField)
	var user User
	json.Unmarshal(userBytes, &user)

	if user.Name != "Alice" {
		t.Errorf("expected user name Alice, got %s", user.Name)
	}
}

func TestJSONSerializerWithExpand(t *testing.T) {
	serializer := &JSONSerializer{}

	data := TestModel{
		ID:    "1",
		Title: "Test Item",
	}

	expand := []string{"author", "tags"}
	result, err := serializer.SerializeWithExpand(data, "/testmodels", expand)
	if err != nil {
		t.Fatalf("JSON serialization with expand failed: %v", err)
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
}

func TestJSONLDSerializerWrapWithContext(t *testing.T) {
	serializer := &JSONLDSerializer{}

	data := TestModel{
		ID:    "1",
		Title: "Test Item",
	}

	wrapped := serializer.wrapWithContext(data, "/testmodels")
	if wrapped["@context"] != "https://schema.org/" {
		t.Errorf("Expected @context='https://schema.org/', got %v", wrapped["@context"])
	}

	if wrapped["@type"] != "TestModel" {
		t.Errorf("Expected @type='TestModel', got %v", wrapped["@type"])
	}

	if wrapped["@id"] != "/testmodels/1" {
		t.Errorf("Expected @id='/testmodels/1', got %v", wrapped["@id"])
	}
}

func TestAddTypeToItemExported(t *testing.T) {
	serializer := &JSONLDSerializer{}
	data := TestModel{
		ID:    "1",
		Title: "Test Item",
	}

	result := serializer.AddTypeToItemExported(data, "/testmodels")

	if result["@type"] != "TestModel" {
		t.Errorf("Expected @type='TestModel', got %v", result["@type"])
	}

	if result["@id"] != "/testmodels/1" {
		t.Errorf("Expected @id='/testmodels/1', got %v", result["@id"])
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user", "users"},
		{"post", "posts"},
		{"category", "categories"},
		{"city", "cities"},
		{"box", "boxes"},
		{"class", "classes"},
		{"wish", "wishes"},
		{"buzz", "buzzes"},
		{"person", "persons"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := pluralize(tt.input)
			if result != tt.expected {
				t.Errorf("pluralize(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsVowel(t *testing.T) {
	vowels := []byte{'a', 'e', 'i', 'o', 'u'}
	consonants := []byte{'b', 'c', 'd', 'f', 'g', 'h', 'j', 'k', 'l', 'm', 'n', 'p', 'q', 'r', 's', 't', 'v', 'w', 'x', 'y', 'z'}

	for _, v := range vowels {
		if !isVowel(v) {
			t.Errorf("Expected %c to be a vowel", v)
		}
	}

	for _, c := range consonants {
		if isVowel(c) {
			t.Errorf("Expected %c not to be a vowel", c)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"firstName", "first_name"},
		{"lastName", "last_name"},
		{"CreatedAt", "created_at"},
		{"UpdatedAt", "updated_at"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user_id", "userId"},
		{"post_id", "postId"},
		{"first_name", "firstName"},
		{"last_name", "lastName"},
		{"created_at", "createdAt"},
		{"updated_at", "updatedAt"},
		{"id", "id"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("toCamelCase(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestJSONLDSerializerWithExpand(t *testing.T) {
	serializer := &JSONLDSerializer{}

	data := TestModel{
		ID:    "1",
		Title: "Test Item",
	}

	expand := []string{"author"}
	result, err := serializer.SerializeWithExpand(data, "/testmodels", expand)
	if err != nil {
		t.Fatalf("JSON-LD serialization with expand failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatalf("Failed to decode JSON-LD: %v", err)
	}

	if decoded["@context"] != "https://schema.org/" {
		t.Errorf("Expected @context='https://schema.org/', got %v", decoded["@context"])
	}
}

func TestExpandRelations_NilForeignKey(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	postCRUD := crud.New[Post](db)
	userCRUD := crud.New[User](db)

	ctx := context.Background()

	_, err := db.Exec(ctx, `INSERT INTO posts (id, user_id, title, content) VALUES ('post-2', NULL, 'Orphan Post', 'No user')`)
	if err != nil {
		t.Fatalf("failed to insert orphan post: %v", err)
	}

	post, err := postCRUD.GetByID(ctx, "post-2")
	if err != nil {
		t.Fatalf("failed to get post: %v", err)
	}

	configs := map[string]RelationConfig{
		"user": {
			Field:           "user",
			ForeignKeyField: "userId",
			RelatedTable:    "users",
			Fetcher:         crud.RelationFetcher(userCRUD),
		},
	}

	expanded, err := ExpandRelations(ctx, *post, []string{"user"}, configs)
	if err != nil {
		t.Fatalf("failed to expand: %v", err)
	}

	expandedMap, ok := expanded.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}

	if _, hasUser := expandedMap["user"]; hasUser {
		t.Error("user field should not be present when userId is nil")
	}
}

func TestExpandRelations_InvalidRelation(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	postCRUD := crud.New[Post](db)

	ctx := context.Background()
	post, err := postCRUD.GetByID(ctx, "post-1")
	if err != nil {
		t.Fatalf("failed to get post: %v", err)
	}

	configs := map[string]RelationConfig{}

	expanded, err := ExpandRelations(ctx, *post, []string{"nonexistent"}, configs)
	if err != nil {
		t.Fatalf("failed to expand: %v", err)
	}

	expandedMap, ok := expanded.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}

	if expandedMap["id"] != "post-1" {
		t.Errorf("Expected post-1, got %v", expandedMap["id"])
	}
}

func TestExpandRelations_EmptyExpand(t *testing.T) {
	type SimplePost struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}

	post := SimplePost{
		ID:    "1",
		Title: "Test",
	}

	configs := map[string]RelationConfig{}

	expanded, err := ExpandRelations(context.Background(), post, []string{}, configs)
	if err != nil {
		t.Fatalf("failed to expand: %v", err)
	}

	expandedMap, ok := expanded.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}

	if expandedMap["id"] != "1" {
		t.Errorf("Expected id=1, got %v", expandedMap["id"])
	}
}

func TestNormalizeFieldName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"userId", "userid"},
		{"user_id", "userid"},
		{"UserID", "userid"},
		{"  firstName  ", "firstname"},
		{"id", "id"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeFieldName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeFieldName(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestInferSchemaType(t *testing.T) {
	serializer := &JSONLDSerializer{}

	tests := []struct {
		data     interface{}
		expected string
	}{
		{TestModel{ID: "1", Title: "Test"}, "TestModel"},
		{map[string]interface{}{"id": "1"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := serializer.inferSchemaType(tt.data)
			if result != tt.expected {
				t.Errorf("inferSchemaType(%+v) = %s, expected %s", tt.data, result, tt.expected)
			}
		})
	}
}

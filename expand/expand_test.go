package expand

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nicolasbonnici/gorest/crud"
	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
)

type User struct {
	ID   string `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

func (User) TableName() string { return "users" }

type Post struct {
	ID      string  `json:"id" db:"id"`
	UserID  *string `json:"userId" db:"user_id"`
	Title   string  `json:"title" db:"title"`
	Content string  `json:"content" db:"content"`
}

func (Post) TableName() string { return "posts" }

func setupTestDB(t *testing.T) (database.Database, func()) {
	db, err := database.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	ctx := context.Background()

	_, err = db.Exec(ctx, `
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}

	_, err = db.Exec(ctx, `
		CREATE TABLE posts (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			title TEXT NOT NULL,
			content TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create posts table: %v", err)
	}

	_, err = db.Exec(ctx, `INSERT INTO users (id, name) VALUES ('user-1', 'Alice')`)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}

	userID := "user-1"
	_, err = db.Exec(ctx, `INSERT INTO posts (id, user_id, title, content) VALUES ('post-1', ?, 'Test Post', 'Content')`, userID)
	if err != nil {
		t.Fatalf("failed to insert test post: %v", err)
	}

	cleanup := func() {
		db.Close()
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
			CRUD:            userCRUD,
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
			CRUD:            userCRUD,
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

//go:build integration

package internal

import (
	"context"
	"testing"

	"github.com/nicolasbonnici/gorest/gen/models"
	"github.com/nicolasbonnici/gorest/internal/crud"
)

func TestCRUD_Create(t *testing.T) {
	c := crud.New[models.Users](db)
	ctx := context.Background()

	user := models.Users{
		Firstname: "John",
		Lastname:  "Doe",
		Email:     "john.doe@example.com",
		Password:  stringPtr("password123"),
	}

	err := c.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = $1", user.Email).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify user creation: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 user, got %d", count)
	}
}

func TestCRUD_GetAll(t *testing.T) {
	c := crud.New[models.Users](db)
	ctx := context.Background()

	_, err := db.Exec(ctx, `
		INSERT INTO users (firstname, lastname, email, password)
		VALUES
			('Alice', 'Smith', 'alice.getall@example.com', 'pass1'),
			('Bob', 'Jones', 'bob.getall@example.com', 'pass2')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	results, err := c.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to get all users: %v", err)
	}

	if len(results) < 2 {
		t.Errorf("Expected at least 2 users, got %d", len(results))
	}
}

func TestCRUD_GetByID(t *testing.T) {
	c := crud.New[models.Users](db)
	ctx := context.Background()

	var userID string
	err := db.QueryRow(ctx, `
		INSERT INTO users (firstname, lastname, email, password)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "Charlie", "Brown", "charlie@example.com", "pass123").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	user, err := c.GetByID(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get user by ID: %v", err)
	}

	if user.Email != "charlie@example.com" {
		t.Errorf("Expected email charlie@example.com, got %v", user.Email)
	}
}

func TestCRUD_Update(t *testing.T) {
	c := crud.New[models.Users](db)
	ctx := context.Background()

	// Insert test user
	var userID string
	err := db.QueryRow(ctx, `
		INSERT INTO users (firstname, lastname, email, password)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "David", "Wilson", "david@example.com", "pass123").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	updatedUser := models.Users{
		Firstname: "David",
		Lastname:  "Wilson-Updated",
		Email:     "david.updated@example.com",
		Password:  stringPtr("newpass"),
	}

	err = c.Update(ctx, userID, updatedUser)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	var email string
	err = db.QueryRow(ctx, "SELECT email FROM users WHERE id = $1", userID).Scan(&email)
	if err != nil {
		t.Fatalf("Failed to verify update: %v", err)
	}

	if email != "david.updated@example.com" {
		t.Errorf("Expected email david.updated@example.com, got %s", email)
	}
}

func TestCRUD_Delete(t *testing.T) {
	c := crud.New[models.Users](db)
	ctx := context.Background()

	var userID string
	err := db.QueryRow(ctx, `
		INSERT INTO users (firstname, lastname, email, password)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "Emma", "Taylor", "emma@example.com", "pass123").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	err = c.Delete(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE id = $1", userID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify deletion: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 users after deletion, got %d", count)
	}
}

func TestCRUD_TodoModel(t *testing.T) {
	c := crud.New[models.Todo](db)
	ctx := context.Background()

	var userID string
	err := db.QueryRow(ctx, `
		INSERT INTO users (firstname, lastname, email, password)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "Test", "User", "test_todo@example.com", "pass").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	todo := models.Todo{
		UserId:  &userID,
		Title:   "Test Todo",
		Content: "This is a test todo item",
	}

	err = c.Create(ctx, todo)
	if err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM todo WHERE title = $1", todo.Title).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify todo creation: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 todo, got %d", count)
	}
}

func stringPtr(s string) *string {
	return &s
}

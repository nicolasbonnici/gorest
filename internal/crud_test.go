//go:build integration

package internal

import (
	"context"
	"testing"

	"github.com/nicolasbonnici/gorest/internal/crud"
	"github.com/nicolasbonnici/gorest/internal/api/models"
)

func TestCRUD_Create(t *testing.T) {
	cleanupTestDB(t)
	c := crud.New[models.User](db)
	ctx := context.Background()

	user := models.User{
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
	query := "SELECT COUNT(*) FROM users WHERE email = " + db.Dialect().Placeholder(1)
	err = db.QueryRow(ctx, query, user.Email).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify user creation: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 user, got %d", count)
	}
}

func TestCRUD_GetAll(t *testing.T) {
	cleanupTestDB(t)
	c := crud.New[models.User](db)
	ctx := context.Background()

	insertQuery := `INSERT INTO users (firstname, lastname, email, password) VALUES ` +
		`(` + db.Dialect().Placeholder(1) + `, ` + db.Dialect().Placeholder(2) + `, ` + db.Dialect().Placeholder(3) + `, ` + db.Dialect().Placeholder(4) + `), ` +
		`(` + db.Dialect().Placeholder(5) + `, ` + db.Dialect().Placeholder(6) + `, ` + db.Dialect().Placeholder(7) + `, ` + db.Dialect().Placeholder(8) + `)`

	_, err := db.Exec(ctx, insertQuery,
		"Alice", "Smith", "alice.getall@example.com", "pass1",
		"Bob", "Jones", "bob.getall@example.com", "pass2")
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
	cleanupTestDB(t)
	c := crud.New[models.User](db)
	ctx := context.Background()

	var userID string
	var insertQuery string
	if db.Dialect().SupportsReturning() {
		insertQuery = `INSERT INTO users (firstname, lastname, email, password) VALUES (` +
			db.Dialect().Placeholder(1) + `, ` + db.Dialect().Placeholder(2) + `, ` +
			db.Dialect().Placeholder(3) + `, ` + db.Dialect().Placeholder(4) + `) ` +
			db.Dialect().ReturningClause()
		err := db.QueryRow(ctx, insertQuery, "Charlie", "Brown", "charlie@example.com", "pass123").Scan(&userID)
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
	} else {
		insertQuery = `INSERT INTO users (firstname, lastname, email, password) VALUES (` +
			db.Dialect().Placeholder(1) + `, ` + db.Dialect().Placeholder(2) + `, ` +
			db.Dialect().Placeholder(3) + `, ` + db.Dialect().Placeholder(4) + `)`
		res, err := db.Exec(ctx, insertQuery, "Charlie", "Brown", "charlie@example.com", "pass123")
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatalf("Failed to get last insert ID: %v", err)
		}
		userID = string(rune(id))
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
	cleanupTestDB(t)
	c := crud.New[models.User](db)
	ctx := context.Background()

	userID := insertTestUser(t, ctx, "David", "Wilson", "david@example.com", "pass123")

	updatedUser := models.User{
		Firstname: "David",
		Lastname:  "Wilson-Updated",
		Email:     "david.updated@example.com",
		Password:  stringPtr("newpass"),
	}

	err := c.Update(ctx, userID, updatedUser)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	var email string
	query := "SELECT email FROM users WHERE id = " + db.Dialect().Placeholder(1)
	err = db.QueryRow(ctx, query, userID).Scan(&email)
	if err != nil {
		t.Fatalf("Failed to verify update: %v", err)
	}

	if email != "david.updated@example.com" {
		t.Errorf("Expected email david.updated@example.com, got %s", email)
	}
}

func TestCRUD_Delete(t *testing.T) {
	cleanupTestDB(t)
	c := crud.New[models.User](db)
	ctx := context.Background()

	userID := insertTestUser(t, ctx, "Emma", "Taylor", "emma@example.com", "pass123")

	err := c.Delete(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	var count int
	query := "SELECT COUNT(*) FROM users WHERE id = " + db.Dialect().Placeholder(1)
	err = db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify deletion: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 users after deletion, got %d", count)
	}
}

func TestCRUD_TodoModel(t *testing.T) {
	cleanupTestDB(t)
	c := crud.New[models.Todo](db)
	ctx := context.Background()

	userID := insertTestUser(t, ctx, "Test", "User", "test_todo@example.com", "pass")

	todo := models.Todo{
		UserId:  &userID,
		Title:   "Test Todo",
		Content: "This is a test todo item",
	}

	err := c.Create(ctx, todo)
	if err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	var count int
	query := "SELECT COUNT(*) FROM todo WHERE title = " + db.Dialect().Placeholder(1)
	err = db.QueryRow(ctx, query, todo.Title).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify todo creation: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 todo, got %d", count)
	}
}

func insertTestUser(t *testing.T, ctx context.Context, firstname, lastname, email, password string) string {
	t.Helper()
	var userID string

	if db.Dialect().SupportsReturning() {
		query := `INSERT INTO users (firstname, lastname, email, password) VALUES (` +
			db.Dialect().Placeholder(1) + `, ` + db.Dialect().Placeholder(2) + `, ` +
			db.Dialect().Placeholder(3) + `, ` + db.Dialect().Placeholder(4) + `) ` +
			db.Dialect().ReturningClause()
		err := db.QueryRow(ctx, query, firstname, lastname, email, password).Scan(&userID)
		if err != nil {
			t.Fatalf("Failed to insert test user: %v", err)
		}
	} else {
		query := `INSERT INTO users (firstname, lastname, email, password) VALUES (` +
			db.Dialect().Placeholder(1) + `, ` + db.Dialect().Placeholder(2) + `, ` +
			db.Dialect().Placeholder(3) + `, ` + db.Dialect().Placeholder(4) + `)`
		res, err := db.Exec(ctx, query, firstname, lastname, email, password)
		if err != nil {
			t.Fatalf("Failed to insert test user: %v", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatalf("Failed to get last insert ID: %v", err)
		}
		userID = string(rune(id))
	}

	return userID
}

func stringPtr(s string) *string {
	return &s
}

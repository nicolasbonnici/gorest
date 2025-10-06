package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolasbonnici/gorest/internal/models"
)

const testDBURL = "postgres://postgres:postgres@localhost:5433/mydb_test?sslmode=disable"

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	db, err := pgxpool.New(context.Background(), testDBURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Create test tables
	_, err = db.Exec(context.Background(), `
		DROP TABLE IF EXISTS todo CASCADE;
		DROP TABLE IF EXISTS users CASCADE;

		CREATE TABLE users (
			id SERIAL PRIMARY KEY,
			firstname VARCHAR(255) NOT NULL,
			lastname VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP
		);

		CREATE TABLE todo (
			id SERIAL PRIMARY KEY,
			user_id INTEGER REFERENCES users(id),
			title VARCHAR(255) NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return db
}

func cleanupTestDB(t *testing.T, db *pgxpool.Pool) {
	t.Helper()
	db.Close()
}

func TestUsersResource_Create(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	app := fiber.New()
	RegisterUsersRoutes(app, db)

	user := models.Users{
		Firstname: "John",
		Lastname:  "Doe",
		Email:     "john@example.com",
		Password:  stringPtr("password123"),
	}

	body, _ := json.Marshal(user)
	req := httptest.NewRequest("POST", "/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != 201 {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	// Verify user was created in database
	var count int
	err = db.QueryRow(context.Background(), "SELECT COUNT(*) FROM users WHERE email = $1", user.Email).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify user creation: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 user in database, got %d", count)
	}
}

func TestUsersResource_List(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Insert test data
	_, err := db.Exec(context.Background(), `
		INSERT INTO users (firstname, lastname, email, password)
		VALUES
			('Alice', 'Smith', 'alice@example.com', 'pass1'),
			('Bob', 'Jones', 'bob@example.com', 'pass2')
	`)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	app := fiber.New()
	RegisterUsersRoutes(app, db)

	req := httptest.NewRequest("GET", "/users", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse response
	body, _ := io.ReadAll(resp.Body)
	var users []models.Users
	err = json.Unmarshal(body, &users)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
}

func TestUsersResource_Get(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Insert test user
	var userID int
	err := db.QueryRow(context.Background(), `
		INSERT INTO users (firstname, lastname, email, password)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "Charlie", "Brown", "charlie@example.com", "pass123").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	app := fiber.New()
	RegisterUsersRoutes(app, db)

	req := httptest.NewRequest("GET", "/users/"+string(rune(userID+48)), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 200, got %d. Body: %s", resp.StatusCode, string(body))
	}
}

func TestUsersResource_Update(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Insert test user
	var userID int
	err := db.QueryRow(context.Background(), `
		INSERT INTO users (firstname, lastname, email, password)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "David", "Wilson", "david@example.com", "pass123").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	app := fiber.New()
	RegisterUsersRoutes(app, db)

	updatedUser := models.Users{
		Firstname: "David",
		Lastname:  "Wilson-Updated",
		Email:     "david.updated@example.com",
		Password:  stringPtr("newpass"),
	}

	body, _ := json.Marshal(updatedUser)
	req := httptest.NewRequest("PUT", "/users/"+string(rune(userID+48)), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify update in database
	var email string
	err = db.QueryRow(context.Background(), "SELECT email FROM users WHERE id = $1", userID).Scan(&email)
	if err != nil {
		t.Fatalf("Failed to verify update: %v", err)
	}

	if email != "david.updated@example.com" {
		t.Errorf("Expected email to be updated to david.updated@example.com, got %s", email)
	}
}

func TestUsersResource_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Insert test user
	var userID int
	err := db.QueryRow(context.Background(), `
		INSERT INTO users (firstname, lastname, email, password)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "Emma", "Taylor", "emma@example.com", "pass123").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	app := fiber.New()
	RegisterUsersRoutes(app, db)

	req := httptest.NewRequest("DELETE", "/users/"+string(rune(userID+48)), nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != 204 {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}

	// Verify deletion in database
	var count int
	err = db.QueryRow(context.Background(), "SELECT COUNT(*) FROM users WHERE id = $1", userID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify deletion: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected user to be deleted, but still exists in database")
	}
}

func TestTodoResource_Create(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// First create a user (foreign key requirement)
	var userID int
	err := db.QueryRow(context.Background(), `
		INSERT INTO users (firstname, lastname, email, password)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "Test", "User", "test@example.com", "pass").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	app := fiber.New()
	RegisterTodoRoutes(app, db)

	todo := models.Todo{
		UserId:  userID,
		Title:   "Test Todo",
		Content: "This is a test todo",
	}

	body, _ := json.Marshal(todo)
	req := httptest.NewRequest("POST", "/todo", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 201, got %d. Body: %s", resp.StatusCode, string(respBody))
	}

	// Verify todo was created in database
	var count int
	err = db.QueryRow(context.Background(), "SELECT COUNT(*) FROM todo WHERE title = $1", todo.Title).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify todo creation: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 todo in database, got %d", count)
	}
}

func TestTodoResource_List(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create test user
	var userID int
	err := db.QueryRow(context.Background(), `
		INSERT INTO users (firstname, lastname, email, password)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "Test", "User", "test@example.com", "pass").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Insert test todos
	_, err = db.Exec(context.Background(), `
		INSERT INTO todo (user_id, title, content)
		VALUES
			($1, 'Todo 1', 'Content 1'),
			($1, 'Todo 2', 'Content 2')
	`, userID)
	if err != nil {
		t.Fatalf("Failed to insert test todos: %v", err)
	}

	app := fiber.New()
	RegisterTodoRoutes(app, db)

	req := httptest.NewRequest("GET", "/todo", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Parse response
	body, _ := io.ReadAll(resp.Body)
	var todos []models.Todo
	err = json.Unmarshal(body, &todos)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(todos) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(todos))
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}

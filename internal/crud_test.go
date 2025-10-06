package internal

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nicolasbonnici/gorest/internal/models"
)

// Test database connection string
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

func TestCRUD_Create(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	crud := New[models.Users](db)
	ctx := context.Background()

	user := models.Users{
		Firstname: "John",
		Lastname:  "Doe",
		Email:     "john.doe@example.com",
		Password:  stringPtr("password123"),
	}

	err := crud.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Verify user was created
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
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	crud := New[models.Users](db)
	ctx := context.Background()

	// Insert test data with unique emails
	users := []models.Users{
		{Firstname: "Alice", Lastname: "Smith", Email: "alice.getall@example.com", Password: stringPtr("pass1")},
		{Firstname: "Bob", Lastname: "Jones", Email: "bob.getall@example.com", Password: stringPtr("pass2")},
	}

	for _, user := range users {
		err := crud.Create(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
	}

	// Test GetAll
	results, err := crud.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to get all users: %v", err)
	}

	if len(results) < 2 {
		t.Errorf("Expected at least 2 users, got %d", len(results))
	}
}

func TestCRUD_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	crud := New[models.Users](db)
	ctx := context.Background()

	// Insert test user and get ID
	var userID int
	err := db.QueryRow(ctx, `
		INSERT INTO users (firstname, lastname, email, password)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "Charlie", "Brown", "charlie@example.com", "pass123").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Test GetByID
	user, err := crud.GetByID(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get user by ID: %v", err)
	}

	if user.Email != "charlie@example.com" {
		t.Errorf("Expected email charlie@example.com, got %s", user.Email)
	}
}

func TestCRUD_Update(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	crud := New[models.Users](db)
	ctx := context.Background()

	// Insert test user
	var userID int
	err := db.QueryRow(ctx, `
		INSERT INTO users (firstname, lastname, email, password)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "David", "Wilson", "david@example.com", "pass123").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Update user
	updatedUser := models.Users{
		Firstname: "David",
		Lastname:  "Wilson-Updated",
		Email:     "david.updated@example.com",
		Password:  stringPtr("newpass"),
	}

	err = crud.Update(ctx, userID, updatedUser)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Verify update
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
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	crud := New[models.Users](db)
	ctx := context.Background()

	// Insert test user
	var userID int
	err := db.QueryRow(ctx, `
		INSERT INTO users (firstname, lastname, email, password)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "Emma", "Taylor", "emma@example.com", "pass123").Scan(&userID)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Delete user
	err = crud.Delete(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// Verify deletion
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

	crud := New[models.Todo](db)
	ctx := context.Background()

	todo := models.Todo{
		UserId:  userID,
		Title:   "Test Todo",
		Content: "This is a test todo item",
	}

	err = crud.Create(ctx, todo)
	if err != nil {
		t.Fatalf("Failed to create todo: %v", err)
	}

	// Verify todo was created
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM todo WHERE title = $1", todo.Title).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify todo creation: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 todo, got %d", count)
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}

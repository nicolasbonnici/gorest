package fixtures

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/nicolasbonnici/gorest/database"
	_ "github.com/nicolasbonnici/gorest/database/sqlite"
)

// User is a test model for fixtures
type User struct {
	ID        string `db:"id"`
	Firstname string `db:"firstname"`
	Lastname  string `db:"lastname"`
	Email     string `db:"email"`
	Password  string `db:"password"`
	UpdatedAt string `db:"updated_at"`
	CreatedAt string `db:"created_at"`
}

func (u User) TableName() string {
	return "users"
}

// Todo is a test model for fixtures
type Todo struct {
	ID        string `db:"id"`
	UserID    string `db:"user_id"`
	Title     string `db:"title"`
	Content   string `db:"content"`
	UpdatedAt string `db:"updated_at"`
	CreatedAt string `db:"created_at"`
}

func (t Todo) TableName() string {
	return "todo"
}

func setupTestDB(t *testing.T) database.Database {
	t.Helper()

	db, err := database.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		db.Close()
		t.Fatalf("database ping failed: %v", err)
	}

	schema := `
		DROP TABLE IF EXISTS todo;
		DROP TABLE IF EXISTS users;

		CREATE TABLE users (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			firstname TEXT NOT NULL,
			lastname TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT,
			updated_at TEXT,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE todo (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			user_id TEXT,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			updated_at TEXT,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
	`

	_, err = db.Exec(ctx, schema)
	if err != nil {
		db.Close()
		t.Fatalf("failed to create schema: %v", err)
	}

	return db
}

func TestNew(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	if loader == nil {
		t.Fatal("expected non-nil loader")
	}

	if loader.db != db {
		t.Error("expected db to be set")
	}

	if loader.loaded == nil {
		t.Error("expected loaded map to be initialized")
	}

	if loader.cleanup {
		t.Error("expected cleanup to be false by default")
	}
}

func TestWithContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	loader := New(db).WithContext(ctx)

	if loader.ctx != ctx {
		t.Error("expected context to be set")
	}
}

func TestWithTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	_, err := loader.WithTransaction()
	if err != nil {
		t.Fatalf("failed to start transaction: %v", err)
	}

	if loader.tx == nil {
		t.Error("expected transaction to be set")
	}

	err = loader.Rollback()
	if err != nil {
		t.Errorf("failed to rollback: %v", err)
	}
}

func TestWithTransaction_AlreadyStarted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	_, err := loader.WithTransaction()
	if err != nil {
		t.Fatalf("failed to start first transaction: %v", err)
	}

	_, err = loader.WithTransaction()
	if err == nil {
		t.Error("expected error when starting second transaction")
	}

	loader.Rollback()
}

func TestCommit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	_, err := loader.WithTransaction()
	if err != nil {
		t.Fatalf("failed to start transaction: %v", err)
	}

	users := []User{
		{ID: "test-1", Firstname: "John", Lastname: "Doe", Email: "john@test.com"},
	}

	_, err = Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load fixtures: %v", err)
	}

	err = loader.Commit()
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	if loader.tx != nil {
		t.Error("expected transaction to be nil after commit")
	}

	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}
}

func TestCommit_NoTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	err := loader.Commit()
	if err == nil {
		t.Error("expected error when committing without transaction")
	}
}

func TestRollback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	_, err := loader.WithTransaction()
	if err != nil {
		t.Fatalf("failed to start transaction: %v", err)
	}

	users := []User{
		{ID: "test-1", Firstname: "John", Lastname: "Doe", Email: "john@test.com"},
	}

	_, err = Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load fixtures: %v", err)
	}

	err = loader.Rollback()
	if err != nil {
		t.Fatalf("failed to rollback: %v", err)
	}

	if loader.tx != nil {
		t.Error("expected transaction to be nil after rollback")
	}

	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 users after rollback, got %d", count)
	}
}

func TestRollback_NoTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	err := loader.Rollback()
	if err == nil {
		t.Error("expected error when rolling back without transaction")
	}
}

func TestLoad_Users(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	users := []User{
		{ID: "user-1", Firstname: "Alice", Lastname: "Smith", Email: "alice@test.com"},
		{ID: "user-2", Firstname: "Bob", Lastname: "Jones", Email: "bob@test.com"},
	}

	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	loaded, ok := loader.Get("users")
	if !ok {
		t.Fatal("expected users to be loaded")
	}

	if len(loaded) != 2 {
		t.Errorf("expected 2 users, got %d", len(loaded))
	}

	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 users in database, got %d", count)
	}
}

func TestLoad_Todos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)

	users := []User{
		{ID: "user-1", Firstname: "Alice", Lastname: "Smith", Email: "alice@test.com"},
	}
	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	todos := []Todo{
		{ID: "todo-1", UserID: "user-1", Title: "Task 1", Content: "Content 1"},
		{ID: "todo-2", UserID: "user-1", Title: "Task 2", Content: "Content 2"},
	}

	_, err = Load(loader, "todos", todos)
	if err != nil {
		t.Fatalf("failed to load todos: %v", err)
	}

	loaded, ok := loader.Get("todos")
	if !ok {
		t.Fatal("expected todos to be loaded")
	}

	if len(loaded) != 2 {
		t.Errorf("expected 2 todos, got %d", len(loaded))
	}

	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM todo").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count todos: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 todos in database, got %d", count)
	}
}

func TestLoad_EmptySlice(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	users := []User{}

	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load empty users: %v", err)
	}

	loaded, ok := loader.Get("users")
	if !ok {
		t.Fatal("expected users to be tracked even if empty")
	}

	if len(loaded) != 0 {
		t.Errorf("expected 0 users, got %d", len(loaded))
	}
}

func TestLoad_WithTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	_, err := loader.WithTransaction()
	if err != nil {
		t.Fatalf("failed to start transaction: %v", err)
	}
	defer loader.Rollback()

	users := []User{
		{ID: "user-1", Firstname: "Charlie", Lastname: "Brown", Email: "charlie@test.com"},
	}

	_, err = Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users in transaction: %v", err)
	}

	loaded, ok := loader.Get("users")
	if !ok {
		t.Fatal("expected users to be loaded")
	}

	if len(loaded) != 1 {
		t.Errorf("expected 1 user, got %d", len(loaded))
	}
}

func TestGetTyped_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	users := []User{
		{ID: "user-1", Firstname: "David", Lastname: "Wilson", Email: "david@test.com"},
	}

	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	typed, err := GetTyped[User](loader, "users")
	if err != nil {
		t.Fatalf("failed to get typed users: %v", err)
	}

	if len(typed) != 1 {
		t.Errorf("expected 1 user, got %d", len(typed))
	}

	if typed[0].Firstname != "David" {
		t.Errorf("expected firstname David, got %s", typed[0].Firstname)
	}
}

func TestGetTyped_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)

	_, err := GetTyped[User](loader, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent fixtures")
	}
}

func TestGetTyped_WrongType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	users := []User{
		{ID: "user-1", Firstname: "Eve", Lastname: "Taylor", Email: "eve@test.com"},
	}

	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	_, err = GetTyped[Todo](loader, "users")
	if err == nil {
		t.Error("expected error when getting wrong type")
	}
}

func TestEnableCleanup(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	if loader.ShouldCleanup() {
		t.Error("expected cleanup to be false initially")
	}

	loader.EnableCleanup()
	if !loader.ShouldCleanup() {
		t.Error("expected cleanup to be true after enabling")
	}
}

func TestGetLoadedFixtures(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	users := []User{
		{ID: "user-1", Firstname: "Frank", Lastname: "Miller", Email: "frank@test.com"},
	}

	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	todos := []Todo{
		{ID: "todo-1", UserID: "user-1", Title: "Task", Content: "Content"},
	}

	_, err = Load(loader, "todos", todos)
	if err != nil {
		t.Fatalf("failed to load todos: %v", err)
	}

	names := loader.GetLoadedFixtures()
	if len(names) != 2 {
		t.Errorf("expected 2 fixture names, got %d", len(names))
	}

	found := make(map[string]bool)
	for _, name := range names {
		found[name] = true
	}

	if !found["users"] || !found["todos"] {
		t.Error("expected to find both users and todos")
	}
}

func TestLoadFromYAML(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "users.yaml")

	yamlContent := `- id: yaml-user-1
  firstname: Grace
  lastname: Hopper
  email: grace@test.com
  password: ""
  updated_at: ""
  created_at: "2024-01-01"
- id: yaml-user-2
  firstname: Ada
  lastname: Lovelace
  email: ada@test.com
  password: ""
  updated_at: ""
  created_at: "2024-01-01"
`

	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("failed to write YAML file: %v", err)
	}

	loader := New(db)
	var users []User

	_, err = loader.LoadFromYAML("users", yamlFile, &users)
	if err != nil {
		t.Fatalf("failed to load from YAML: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}

	loaded, ok := loader.Get("users")
	if !ok {
		t.Fatal("expected users to be loaded")
	}

	if len(loaded) != 2 {
		t.Errorf("expected 2 loaded users, got %d", len(loaded))
	}
}

func TestLoadFromYAML_FileNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	var users []User

	_, err := loader.LoadFromYAML("users", "/nonexistent/file.yaml", &users)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadFromYAML_InvalidYAML(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "invalid.yaml")

	err := os.WriteFile(yamlFile, []byte("invalid: yaml: content:"), 0644)
	if err != nil {
		t.Fatalf("failed to write invalid YAML file: %v", err)
	}

	loader := New(db)
	var users []User

	_, err = loader.LoadFromYAML("users", yamlFile, &users)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadFromJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "users.json")

	jsonContent := `[
		{
			"id": "json-user-1",
			"firstname": "Isaac",
			"lastname": "Newton",
			"email": "isaac@test.com",
			"password": "",
			"updated_at": "",
			"created_at": "2024-01-01"
		}
	]`

	err := os.WriteFile(jsonFile, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("failed to write JSON file: %v", err)
	}

	loader := New(db)
	var users []User

	_, err = loader.LoadFromJSON("users", jsonFile, &users)
	if err != nil {
		t.Fatalf("failed to load from JSON: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}

	loaded, ok := loader.Get("users")
	if !ok {
		t.Fatal("expected users to be loaded")
	}

	if len(loaded) != 1 {
		t.Errorf("expected 1 loaded user, got %d", len(loaded))
	}
}

func TestLoadFromJSON_FileNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	var users []User

	_, err := loader.LoadFromJSON("users", "/nonexistent/file.json", &users)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadFromJSON_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "invalid.json")

	err := os.WriteFile(jsonFile, []byte("{invalid json}"), 0644)
	if err != nil {
		t.Fatalf("failed to write invalid JSON file: %v", err)
	}

	loader := New(db)
	var users []User

	_, err = loader.LoadFromJSON("users", jsonFile, &users)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestCleanup_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()
	users := []User{
		{ID: "cleanup-1", Firstname: "Jane", Lastname: "Doe", Email: "jane@test.com"},
		{ID: "cleanup-2", Firstname: "John", Lastname: "Smith", Email: "john@test.com"},
	}

	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 users before cleanup, got %d", count)
	}

	err = Cleanup(loader)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users after cleanup: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 users after cleanup, got %d", count)
	}

	if len(loader.loaded) != 0 {
		t.Error("expected loaded map to be cleared")
	}
}

func TestCleanup_Disabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	users := []User{
		{ID: "no-cleanup-1", Firstname: "Keep", Lastname: "Me", Email: "keep@test.com"},
	}

	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	err = Cleanup(loader)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 user (cleanup disabled), got %d", count)
	}
}

func TestCleanup_WithForeignKeys(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()

	users := []User{
		{ID: "fk-user-1", Firstname: "Parent", Lastname: "User", Email: "parent@test.com"},
	}
	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	todos := []Todo{
		{ID: "fk-todo-1", UserID: "fk-user-1", Title: "Child Task", Content: "Content"},
	}
	_, err = Load(loader, "todos", todos)
	if err != nil {
		t.Fatalf("failed to load todos: %v", err)
	}

	err = CleanupOrdered(loader, []string{"users", "todos"})
	if err != nil {
		t.Fatalf("ordered cleanup failed: %v", err)
	}

	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 users after cleanup, got %d", count)
	}

	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM todo").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count todos: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 todos after cleanup, got %d", count)
	}
}

func TestCleanup_Rollback(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()
	_, err := loader.WithTransaction()
	if err != nil {
		t.Fatalf("failed to start transaction: %v", err)
	}

	users := []User{
		{ID: "rollback-1", Firstname: "Rollback", Lastname: "User", Email: "rollback@test.com"},
	}
	_, err = Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	err = CleanupWithStrategy(loader, CleanupRollback)
	if err != nil {
		t.Fatalf("rollback cleanup failed: %v", err)
	}

	if loader.tx != nil {
		t.Error("expected transaction to be nil after rollback cleanup")
	}

	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 users after rollback, got %d", count)
	}
}

func TestCleanup_Rollback_NoTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()

	err := CleanupWithStrategy(loader, CleanupRollback)
	if err == nil {
		t.Error("expected error when rolling back without transaction")
	}
}

func TestCleanup_Truncate(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()
	users := []User{
		{ID: "truncate-1", Firstname: "Truncate", Lastname: "User", Email: "truncate@test.com"},
	}

	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	err = CleanupWithStrategy(loader, CleanupTruncate)
	if err != nil {
		t.Fatalf("truncate cleanup failed: %v", err)
	}

	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 users after truncate, got %d", count)
	}
}

func TestNewBuilder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)
	if builder == nil {
		t.Fatal("expected non-nil builder")
	}

	if builder.loader == nil {
		t.Error("expected loader to be set")
	}

	if builder.t != nil {
		t.Error("expected t to be nil")
	}
}

func TestNewBuilderWithT(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilderWithT(t, db)
	if builder == nil {
		t.Fatal("expected non-nil builder")
	}

	if builder.loader == nil {
		t.Error("expected loader to be set")
	}

	if builder.t != t {
		t.Error("expected t to be set")
	}
}

func TestBuilder_Load(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)
	users := []User{
		{ID: "builder-1", Firstname: "Builder", Lastname: "Test", Email: "builder@test.com"},
	}

	LoadBuilder(builder, "users", users)

	if builder.Error() != nil {
		t.Fatalf("expected no error, got: %v", builder.Error())
	}

	loaded, ok := builder.Get("users")
	if !ok {
		t.Fatal("expected users to be loaded")
	}

	if len(loaded) != 1 {
		t.Errorf("expected 1 user, got %d", len(loaded))
	}
}

func TestBuilder_Chaining(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	users := []User{
		{ID: "chain-user-1", Firstname: "Chain", Lastname: "User", Email: "chain@test.com"},
	}

	todos := []Todo{
		{ID: "chain-todo-1", UserID: "chain-user-1", Title: "Task", Content: "Content"},
	}

	builder := NewBuilder(db)
	LoadBuilder(builder, "users", users)
	LoadBuilder(builder, "todos", todos)
	builder.Cleanup()

	if builder.Error() != nil {
		t.Fatalf("expected no error, got: %v", builder.Error())
	}

	if !builder.loader.ShouldCleanup() {
		t.Error("expected cleanup to be enabled")
	}

	userList, ok := builder.Get("users")
	if !ok || len(userList) != 1 {
		t.Error("expected users to be loaded")
	}

	todoList, ok := builder.Get("todos")
	if !ok || len(todoList) != 1 {
		t.Error("expected todos to be loaded")
	}
}

func TestBuilder_WithTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db).WithTransaction()

	if builder.Error() != nil {
		t.Fatalf("expected no error, got: %v", builder.Error())
	}

	if builder.loader.tx == nil {
		t.Error("expected transaction to be set")
	}

	builder.Rollback()
}

func TestBuilder_GetTyped(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)
	users := []User{
		{ID: "typed-1", Firstname: "Typed", Lastname: "User", Email: "typed@test.com"},
	}

	LoadBuilder(builder, "users", users)

	result, err := GetTypedFromBuilder[User](builder, "users")
	if err != nil {
		t.Fatalf("failed to get typed users: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 user, got %d", len(result))
	}

	if result[0].Firstname != "Typed" {
		t.Errorf("expected firstname Typed, got %s", result[0].Firstname)
	}
}

func TestBuilder_ErrorPropagation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)
	builder.err = fmt.Errorf("simulated error")

	LoadBuilder(builder, "users", []User{})

	if builder.Error() == nil {
		t.Error("expected error to be propagated")
	}
}

func TestBuilder_WithContext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	builder := NewBuilder(db).WithContext(ctx)

	if builder.loader.ctx != ctx {
		t.Error("expected context to be set")
	}
}

func TestBuilder_Commit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db).WithTransaction()
	users := []User{
		{ID: "commit-1", Firstname: "Commit", Lastname: "User", Email: "commit@test.com"},
	}

	LoadBuilder(builder, "users", users).Commit()

	if builder.Error() != nil {
		t.Fatalf("expected no error, got: %v", builder.Error())
	}

	ctx := context.Background()
	var count int
	err := db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 user after commit, got %d", count)
	}
}

func TestBuilder_Rollback_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db).WithTransaction()
	users := []User{
		{ID: "rollback-1", Firstname: "Rollback", Lastname: "User", Email: "rollback@test.com"},
	}

	LoadBuilder(builder, "users", users).Rollback()

	if builder.Error() != nil {
		t.Fatalf("expected no error, got: %v", builder.Error())
	}

	ctx := context.Background()
	var count int
	err := db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 users after rollback, got %d", count)
	}
}

func TestBuilder_LoadFromYAML(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "users.yaml")

	yamlContent := `- id: yaml-builder-1
  firstname: YAML
  lastname: Builder
  email: yaml-builder@test.com
  password: ""
  updated_at: ""
  created_at: "2024-01-01"
`

	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("failed to write YAML file: %v", err)
	}

	var users []User
	builder := NewBuilder(db).LoadFromYAML("users", yamlFile, &users)

	if builder.Error() != nil {
		t.Fatalf("expected no error, got: %v", builder.Error())
	}

	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
}

func TestBuilder_LoadFromJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	jsonFile := filepath.Join(tmpDir, "users.json")

	jsonContent := `[{"id": "json-builder-1", "firstname": "JSON", "lastname": "Builder", "email": "json-builder@test.com"}]`

	err := os.WriteFile(jsonFile, []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("failed to write JSON file: %v", err)
	}

	var users []User
	builder := NewBuilder(db).LoadFromJSON("users", jsonFile, &users)

	if builder.Error() != nil {
		t.Fatalf("expected no error, got: %v", builder.Error())
	}

	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
}

func TestBuilder_Loader(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)
	loader := builder.Loader()

	if loader == nil {
		t.Error("expected loader to be returned")
	}

	if loader != builder.loader {
		t.Error("expected same loader instance")
	}
}

func TestBuilder_GetTyped_Error(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)
	builder.err = fmt.Errorf("test error")

	_, err := GetTypedFromBuilder[User](builder, "users")
	if err == nil {
		t.Error("expected error to be returned")
	}
}

func TestBuilder_GetTyped_UnsupportedType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)

	// This test is no longer valid with generic API - type mismatches are compile-time errors now
	// Testing that we can get a valid type successfully
	LoadBuilder(builder, "users", []User{{ID: "test", Email: "test@example.com", Firstname: "Test", Lastname: "User"}})
	_, err := GetTypedFromBuilder[User](builder, "users")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBuilder_Load_UnsupportedType(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)

	// This test is no longer valid with generic API - non-crud.Model types won't compile
	// The type constraint crud.Model prevents this at compile time
	// Testing valid loading instead
	users := []User{{ID: "test", Email: "test@example.com", Firstname: "Test", Lastname: "User"}}
	LoadBuilder(builder, "users", users)

	if builder.Error() != nil {
		t.Errorf("unexpected error: %v", builder.Error())
	}
}

func TestGetTableName_WithMethod(t *testing.T) {
	type CustomModel struct {
		ID string
	}

	model := CustomModel{ID: "test"}
	tableName := getTableName(model)

	if tableName != "" {
		t.Errorf("expected empty table name for non-Model type, got %s", tableName)
	}
}

func TestGetIDValue_NotStruct(t *testing.T) {
	id := getIDValue("not a struct")
	if id != nil {
		t.Errorf("expected nil for non-struct, got %v", id)
	}
}

func TestCleanupOrdered_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()

	err := CleanupOrdered(loader, []string{"nonexistent"})
	if err != nil {
		t.Errorf("expected no error for nonexistent fixtures, got: %v", err)
	}
}

func TestCleanup_EmptyFixtures(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()
	loader.loaded["test"] = []interface{}{}

	err := Cleanup(loader)
	if err != nil {
		t.Errorf("expected no error for empty fixtures, got: %v", err)
	}
}

func TestInsertFixture_WithoutID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	user := User{Firstname: "NoID", Lastname: "User", Email: "noid@test.com"}

	err := loader.insertFixture(user)
	if err != nil {
		t.Fatalf("failed to insert fixture without ID: %v", err)
	}

	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = ?", "noid@test.com").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}
}

// joinStrings tests removed - now using strings.Join from standard library

func BenchmarkLoad(b *testing.B) {
	db, err := database.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	schema := `
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			firstname TEXT NOT NULL,
			lastname TEXT NOT NULL,
			email TEXT UNIQUE NOT NULL,
			password TEXT,
			updated_at TEXT,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err = db.Exec(ctx, schema)
	if err != nil {
		b.Fatalf("failed to create schema: %v", err)
	}

	users := []User{
		{ID: "bench-1", Firstname: "Bench", Lastname: "User", Email: "bench@test.com"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loader := New(db)
		Load(loader, "users", users)
		db.Exec(ctx, "DELETE FROM users")
	}
}

func TestBuilder_Cleanup(t *testing.T) {
	db := setupTestDB(t)

	subtest := func(tt *testing.T) {
		builder := NewBuilderWithT(tt, db)
		users := []User{
			{ID: "cleanup-builder-1", Firstname: "Cleanup", Lastname: "Test", Email: "cleanup@test.com"},
		}

		LoadBuilder(builder, "users", users).Cleanup()

		if !builder.loader.ShouldCleanup() {
			tt.Error("expected cleanup to be enabled")
		}
	}

	t.Run("cleanup", subtest)

	ctx := context.Background()
	var count int
	err := db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users after subtest: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 users after cleanup, got %d", count)
	}

	db.Close()
}

func TestBuilder_WithTransaction_Error(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)
	builder.err = fmt.Errorf("existing error")

	builder.WithTransaction()

	if builder.Error().Error() != "existing error" {
		t.Error("expected existing error to be preserved")
	}
}

func TestBuilder_LoadFromYAML_Error(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)
	builder.err = fmt.Errorf("existing error")

	var users []User
	builder.LoadFromYAML("users", "nonexistent.yaml", &users)

	if builder.Error().Error() != "existing error" {
		t.Error("expected existing error to be preserved")
	}
}

func TestBuilder_LoadFromJSON_Error(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)
	builder.err = fmt.Errorf("existing error")

	var users []User
	builder.LoadFromJSON("users", "nonexistent.json", &users)

	if builder.Error().Error() != "existing error" {
		t.Error("expected existing error to be preserved")
	}
}

func TestBuilder_Commit_Error(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)
	builder.err = fmt.Errorf("existing error")

	builder.Commit()

	if builder.Error().Error() != "existing error" {
		t.Error("expected existing error to be preserved")
	}
}

func TestBuilder_Rollback_Error(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)
	builder.err = fmt.Errorf("existing error")

	builder.Rollback()

	if builder.Error().Error() != "existing error" {
		t.Error("expected existing error to be preserved")
	}
}

func TestBuilder_Cleanup_Error(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)
	builder.err = fmt.Errorf("existing error")

	builder.Cleanup()

	if builder.Error().Error() != "existing error" {
		t.Error("expected existing error to be preserved")
	}
}

func TestBuilder_Load_ErrorPropagation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)
	builder.err = fmt.Errorf("existing error")

	users := []User{{ID: "test", Firstname: "Test", Lastname: "User", Email: "test@test.com"}}
	LoadBuilder(builder, "users", users)

	if builder.Error().Error() != "existing error" {
		t.Error("expected existing error to be preserved")
	}
}

func TestGetTableName_Pointer(t *testing.T) {
	user := &User{ID: "test"}
	tableName := getTableName(user)

	if tableName != "users" {
		t.Errorf("expected 'users', got %s", tableName)
	}
}

func TestGetTableName_WithTableNameMethod(t *testing.T) {
	type ModelWithMethod struct {
		ID string `db:"id"`
	}

	m := ModelWithMethod{ID: "test"}

	v := reflect.ValueOf(m)
	method := v.MethodByName("TableName")
	if method.IsValid() {
		t.Error("expected no TableName method")
	}

	tableName := getTableName(m)
	if tableName != "" {
		t.Errorf("expected empty table name, got %s", tableName)
	}
}

func TestGetIDValue_Pointer(t *testing.T) {
	user := &User{ID: "test-id"}
	id := getIDValue(user)

	if id == nil {
		t.Error("expected non-nil ID")
	}

	if id.(string) != "test-id" {
		t.Errorf("expected 'test-id', got %v", id)
	}
}

func TestDeleteFixtures_WithTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()
	_, err := loader.WithTransaction()
	if err != nil {
		t.Fatalf("failed to start transaction: %v", err)
	}

	users := []User{
		{ID: "tx-delete-1", Firstname: "TX", Lastname: "Delete", Email: "txdelete@test.com"},
	}

	_, err = Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	err = deleteFixtures(loader, loader.loaded["users"])
	if err != nil {
		t.Fatalf("failed to delete fixtures: %v", err)
	}

	loader.Rollback()
}

func TestCleanupWithDelete_AfterQuery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()

	users := []User{
		{ID: "delete-after-1", Firstname: "Delete", Lastname: "After", Email: "deleteafter@test.com"},
	}

	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	err = cleanupWithDelete(loader)
	if err != nil {
		t.Fatalf("cleanupWithDelete failed: %v", err)
	}

	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 users after cleanup, got %d", count)
	}
}

func TestCleanupOrdered_WithData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()

	users := []User{
		{ID: "ordered-1", Firstname: "Ordered", Lastname: "User", Email: "ordered@test.com"},
	}

	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	todos := []Todo{
		{ID: "ordered-todo-1", UserID: "ordered-1", Title: "Task", Content: "Content"},
	}

	_, err = Load(loader, "todos", todos)
	if err != nil {
		t.Fatalf("failed to load todos: %v", err)
	}

	err = CleanupOrdered(loader, []string{"users", "todos"})
	if err != nil {
		t.Fatalf("ordered cleanup failed: %v", err)
	}

	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 users, got %d", count)
	}
}

func TestCleanupWithTruncate_WithTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()
	_, err := loader.WithTransaction()
	if err != nil {
		t.Fatalf("failed to start transaction: %v", err)
	}

	users := []User{
		{ID: "truncate-tx-1", Firstname: "Truncate", Lastname: "TX", Email: "truncatetx@test.com"},
	}

	_, err = Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	err = cleanupWithTruncate(loader)
	if err != nil {
		t.Fatalf("truncate failed: %v", err)
	}

	loader.Rollback()
}

func TestCleanupWithRollback_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()
	_, err := loader.WithTransaction()
	if err != nil {
		t.Fatalf("failed to start transaction: %v", err)
	}

	users := []User{
		{ID: "rollback-cleanup-1", Firstname: "Rollback", Lastname: "Cleanup", Email: "rollbackcleanup@test.com"},
	}

	_, err = Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	err = cleanupWithRollback(loader)
	if err != nil {
		t.Fatalf("rollback cleanup failed: %v", err)
	}

	if loader.tx != nil {
		t.Error("expected transaction to be nil after rollback")
	}
}

func TestDeleteFixtures_NoIDs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()

	fixtures := []interface{}{
		User{ID: "", Firstname: "NoID", Lastname: "User", Email: "noidelete@test.com"},
	}

	err := deleteFixtures(loader, fixtures)
	if err != nil {
		t.Errorf("expected no error when no IDs to delete, got: %v", err)
	}
}

func TestCleanupWithDelete_EmptyLoaded(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()
	loader.loaded = map[string][]interface{}{}

	err := cleanupWithDelete(loader)
	if err != nil {
		t.Fatalf("expected no error for empty loaded, got: %v", err)
	}
}

func TestCleanupWithTruncate_EmptyLoaded(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()
	loader.loaded = map[string][]interface{}{}

	err := cleanupWithTruncate(loader)
	if err != nil {
		t.Fatalf("expected no error for empty loaded, got: %v", err)
	}
}

func TestBuilder_GetTyped_Todos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)

	users := []User{
		{ID: "user-todos-1", Firstname: "User", Lastname: "ForTodos", Email: "userfortodos@test.com"},
	}
	_, err := Load(builder.loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	todos := []Todo{
		{ID: "typed-todo-1", UserID: "user-todos-1", Title: "Typed", Content: "Todo"},
	}
	_, err = Load(builder.loader, "todos", todos)
	if err != nil {
		t.Fatalf("failed to load todos: %v", err)
	}

	result, err := GetTypedFromBuilder[Todo](builder, "todos")
	if err != nil {
		t.Fatalf("failed to get typed todos: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("expected 1 todo, got %d", len(result))
	}
}

func TestBuilder_LoadTyped_Todos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	builder := NewBuilder(db)

	users := []User{
		{ID: "load-typed-user-1", Firstname: "LoadTyped", Lastname: "User", Email: "loadtypeduser@test.com"},
	}
	LoadBuilder(builder, "users", users)

	todos := []Todo{
		{ID: "load-typed-todo-1", UserID: "load-typed-user-1", Title: "LoadTyped", Content: "Todo"},
	}
	LoadBuilder(builder, "todos", todos)

	if builder.Error() != nil {
		t.Fatalf("expected no error, got: %v", builder.Error())
	}

	loaded, ok := builder.Get("todos")
	if !ok || len(loaded) != 1 {
		t.Error("expected todos to be loaded")
	}
}

func TestInsertFixture_InTransaction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	_, err := loader.WithTransaction()
	if err != nil {
		t.Fatalf("failed to start transaction: %v", err)
	}

	user := User{ID: "insert-tx-1", Firstname: "Insert", Lastname: "TX", Email: "inserttx@test.com"}
	err = loader.insertFixture(user)
	if err != nil {
		t.Fatalf("failed to insert fixture in transaction: %v", err)
	}

	loader.Commit()

	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE id = ?", "insert-tx-1").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}
}

func TestDeleteFixtures_NoTableName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()

	type NoTable struct {
		Name string
	}

	fixtures := []interface{}{
		NoTable{Name: "test"},
	}

	err := deleteFixtures(loader, fixtures)
	if err == nil {
		t.Error("expected error when no table name available")
	}
}

func TestCleanupWithDelete_MultipleTables(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()

	users := []User{
		{ID: "multi-1", Firstname: "Multi", Lastname: "User", Email: "multi@test.com"},
	}
	_, err := Load(loader, "users", users)
	if err != nil {
		t.Fatalf("failed to load users: %v", err)
	}

	todos := []Todo{
		{ID: "multi-todo-1", UserID: "multi-1", Title: "Multi", Content: "Todo"},
	}
	_, err = Load(loader, "todos", todos)
	if err != nil {
		t.Fatalf("failed to load todos: %v", err)
	}

	err = cleanupWithDelete(loader)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	ctx := context.Background()
	var count int
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 users, got %d", count)
	}

	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM todo").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count todos: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 todos, got %d", count)
	}
}






func TestUser_TableName(t *testing.T) {
	user := User{}
	if user.TableName() != "users" {
		t.Errorf("expected table name 'users', got %s", user.TableName())
	}
}

func TestTodo_TableName(t *testing.T) {
	todo := Todo{}
	if todo.TableName() != "todo" {
		t.Errorf("expected table name 'todo', got %s", todo.TableName())
	}
}

func TestCleanupWithStrategy_UnknownStrategy(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db).EnableCleanup()

	err := CleanupWithStrategy(loader, CleanupStrategy(999))
	if err == nil {
		t.Error("expected error for unknown strategy")
	}
}

func TestInsertRawData_NotSlice(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	loader := New(db)
	err := loader.insertRawData("test", "not a slice")
	if err == nil {
		t.Error("expected error for non-slice data")
	}
}

func TestGetTableName_NoTableName(t *testing.T) {
	type NoTableName struct {
		ID string
	}

	tableName := getTableName(NoTableName{ID: "test"})
	if tableName != "" {
		t.Errorf("expected empty table name, got %s", tableName)
	}
}

func TestGetIDValue_NoID(t *testing.T) {
	type NoID struct {
		Name string `db:"name"`
	}

	id := getIDValue(NoID{Name: "test"})
	if id != nil {
		t.Errorf("expected nil ID, got %v", id)
	}
}

func TestGetIDValue_ZeroID(t *testing.T) {
	type ZeroID struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}

	id := getIDValue(ZeroID{ID: "", Name: "test"})
	if id != nil {
		t.Errorf("expected nil for zero ID, got %v", id)
	}
}

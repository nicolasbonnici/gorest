package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/database"
	"golang.org/x/crypto/bcrypt"
)

type mockRow struct {
	scanFunc func(dest ...interface{}) error
}

func (m *mockRow) Scan(dest ...interface{}) error {
	return m.scanFunc(dest...)
}

type mockRows struct {
	scanFunc func(dest ...interface{}) error
	nextFunc func() bool
	closeErr error
	rowErr   error
}

func (m *mockRows) Next() bool {
	return m.nextFunc()
}

func (m *mockRows) Scan(dest ...interface{}) error {
	return m.scanFunc(dest...)
}

func (m *mockRows) Close() error {
	return m.closeErr
}

func (m *mockRows) Err() error {
	return m.rowErr
}

type mockResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (m *mockResult) LastInsertId() (int64, error) {
	return m.lastInsertID, nil
}

func (m *mockResult) RowsAffected() (int64, error) {
	return m.rowsAffected, nil
}

type mockDialect struct{}

func (m *mockDialect) Placeholder(n int) string {
	return "?"
}

func (m *mockDialect) SupportsReturning() bool {
	return false
}

func (m *mockDialect) ReturningClause(cols ...string) string {
	return ""
}

func (m *mockDialect) LimitOffset(limit, offset int) string {
	return ""
}

func (m *mockDialect) QuoteIdentifier(name string) string {
	return name
}

func (m *mockDialect) MapType(dbType string) string {
	return dbType
}

func (m *mockDialect) CaseInsensitiveLike() string {
	return "LOWER"
}

type mockDatabase struct {
	queryRowFunc func(ctx context.Context, query string, args ...interface{}) database.Row
	queryFunc    func(ctx context.Context, query string, args ...interface{}) (database.Rows, error)
	execFunc     func(ctx context.Context, query string, args ...interface{}) (database.Result, error)
	pingFunc     func(ctx context.Context) error
}

func (m *mockDatabase) Connect(ctx context.Context, dsn string) error {
	return nil
}

func (m *mockDatabase) Close() error {
	return nil
}

func (m *mockDatabase) Ping(ctx context.Context) error {
	if m.pingFunc != nil {
		return m.pingFunc(ctx)
	}
	return nil
}

func (m *mockDatabase) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query, args...)
	}
	return nil, nil
}

func (m *mockDatabase) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(ctx, query, args...)
	}
	return nil
}

func (m *mockDatabase) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, query, args...)
	}
	return &mockResult{}, nil
}

func (m *mockDatabase) Begin(ctx context.Context) (database.Tx, error) {
	return nil, nil
}

func (m *mockDatabase) Dialect() database.Dialect {
	return &mockDialect{}
}

func (m *mockDatabase) DriverName() string {
	return "mock"
}

func (m *mockDatabase) Introspector() database.SchemaIntrospector {
	return nil
}

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"simple password", "password123"},
		{"complex password", "P@ssw0rd!#$%"},
		{"long password", strings.Repeat("a", 72)},
		{"empty password", ""},
		{"unicode password", "пароль密码🔐"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if hash == "" {
				t.Fatal("expected non-empty hash")
			}

			if hash == tt.password {
				t.Error("hash should not equal plain password")
			}

			err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(tt.password))
			if err != nil {
				t.Errorf("hash verification failed: %v", err)
			}
		})
	}
}

func TestHashPassword_Uniqueness(t *testing.T) {
	password := "testpassword"
	hash1, err1 := HashPassword(password)
	hash2, err2 := HashPassword(password)

	if err1 != nil || err2 != nil {
		t.Fatalf("expected no errors, got %v, %v", err1, err2)
	}

	if hash1 == hash2 {
		t.Error("expected different hashes due to random salt")
	}
}

func TestVerifyPassword_Success(t *testing.T) {
	password := "correctpassword"
	hash, _ := HashPassword(password)

	err := verifyPassword(password, hash)

	if err != nil {
		t.Errorf("expected password verification to succeed, got error: %v", err)
	}
}

func TestVerifyPassword_Failure(t *testing.T) {
	password := "correctpassword"
	wrongPassword := "wrongpassword"
	hash, _ := HashPassword(password)

	err := verifyPassword(wrongPassword, hash)

	if err == nil {
		t.Error("expected password verification to fail")
	}
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	err := verifyPassword("password", "invalid-hash")

	if err == nil {
		t.Error("expected error for invalid hash")
	}
}

func TestAuthPlugin_Name(t *testing.T) {
	plugin := NewPlugin()

	if plugin.Name() != "auth" {
		t.Errorf("expected plugin name 'auth', got '%s'", plugin.Name())
	}
}

func TestAuthPlugin_Initialize(t *testing.T) {
	plugin := &AuthPlugin{}
	db := &mockDatabase{}

	config := map[string]interface{}{
		"jwt_secret": "test-secret",
		"database":   db,
		"jwt_ttl":    3600,
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.jwtSecret != "test-secret" {
		t.Errorf("expected jwt_secret 'test-secret', got '%s'", plugin.jwtSecret)
	}

	if plugin.db != db {
		t.Error("expected database to be set")
	}

	if plugin.jwtTTL != 3600 {
		t.Errorf("expected jwt_ttl 3600, got %d", plugin.jwtTTL)
	}
}

func TestAuthPlugin_Initialize_PartialConfig(t *testing.T) {
	plugin := &AuthPlugin{}

	config := map[string]interface{}{
		"jwt_secret": "partial-secret",
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.jwtSecret != "partial-secret" {
		t.Errorf("expected jwt_secret 'partial-secret', got '%s'", plugin.jwtSecret)
	}

	if plugin.db != nil {
		t.Error("expected database to be nil")
	}

	if plugin.jwtTTL != 0 {
		t.Errorf("expected jwt_ttl 0, got %d", plugin.jwtTTL)
	}
}

func TestAuthPlugin_Initialize_EmptyConfig(t *testing.T) {
	plugin := &AuthPlugin{}
	err := plugin.Initialize(map[string]interface{}{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}


func TestGetAuthenticatedUser_WithoutUser(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		user := GetAuthenticatedUser(c)

		if user != nil {
			t.Error("expected nil user")
		}

		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	app.Test(req)
}


func TestContext_WithoutUser(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		ctx := Context(c)

		userID := ctx.Value("user_id")
		if userID != nil {
			t.Error("expected nil user_id in context")
		}

		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	app.Test(req)
}

func TestSetupAuth_LoginSuccess(t *testing.T) {
	app := fiber.New()
	hashedPassword, _ := HashPassword("correctpassword")

	db := &mockDatabase{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					*dest[0].(*string) = "user123"
					*dest[1].(*string) = hashedPassword
					*dest[2].(*string) = "John"
					*dest[3].(*string) = "Doe"
					return nil
				},
			}
		},
	}

	SetupAuth(app, db, "test-secret", 3600)

	loginBody := `{"email":"test@example.com","password":"correctpassword"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["token"] == nil {
		t.Error("expected token in response")
	}

	if result["user"] == nil {
		t.Error("expected user in response")
	}
}

func TestSetupAuth_LoginInvalidCredentials(t *testing.T) {
	app := fiber.New()
	hashedPassword, _ := HashPassword("correctpassword")

	db := &mockDatabase{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					*dest[0].(*string) = "user123"
					*dest[1].(*string) = hashedPassword
					*dest[2].(*string) = "John"
					*dest[3].(*string) = "Doe"
					return nil
				},
			}
		},
	}

	SetupAuth(app, db, "test-secret", 3600)

	loginBody := `{"email":"test@example.com","password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["error"] != "Invalid credentials" {
		t.Errorf("expected 'Invalid credentials', got '%v'", result["error"])
	}
}

func TestSetupAuth_LoginUserNotFound(t *testing.T) {
	app := fiber.New()

	db := &mockDatabase{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					return errors.New("sql: no rows in result set")
				},
			}
		},
	}

	SetupAuth(app, db, "test-secret", 3600)

	loginBody := `{"email":"nonexistent@example.com","password":"password"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", resp.StatusCode)
	}
}

func TestSetupAuth_LoginMissingEmail(t *testing.T) {
	app := fiber.New()
	db := &mockDatabase{}

	SetupAuth(app, db, "test-secret", 3600)

	loginBody := `{"password":"password"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["error"] != "Email and password are required" {
		t.Errorf("expected 'Email and password are required', got '%v'", result["error"])
	}
}

func TestSetupAuth_LoginMissingPassword(t *testing.T) {
	app := fiber.New()
	db := &mockDatabase{}

	SetupAuth(app, db, "test-secret", 3600)

	loginBody := `{"email":"test@example.com"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestSetupAuth_LoginInvalidJSON(t *testing.T) {
	app := fiber.New()
	db := &mockDatabase{}

	SetupAuth(app, db, "test-secret", 3600)

	loginBody := `{invalid json`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != 400 {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestSetupAuth_RateLimiting(t *testing.T) {
	app := fiber.New()
	hashedPassword, _ := HashPassword("password")

	db := &mockDatabase{
		queryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &mockRow{
				scanFunc: func(dest ...interface{}) error {
					*dest[0].(*string) = "user123"
					*dest[1].(*string) = hashedPassword
					*dest[2].(*string) = "John"
					*dest[3].(*string) = "Doe"
					return nil
				},
			}
		},
	}

	SetupAuth(app, db, "test-secret", 3600)

	loginBody := `{"email":"test@example.com","password":"password"}`

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "127.0.0.1")
		app.Test(req)
	}

	req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	resp, _ := app.Test(req)

	if resp.StatusCode != 429 {
		t.Errorf("expected status 429 (rate limited), got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if !strings.Contains(result["error"].(string), "Too many login attempts") {
		t.Errorf("expected rate limit error, got '%v'", result["error"])
	}
}

func TestAuthPlugin_SetupEndpoints_NilDatabase(t *testing.T) {
	plugin := &AuthPlugin{
		jwtSecret: "test-secret",
		db:        nil,
	}

	app := fiber.New()
	err := plugin.SetupEndpoints(app)

	if err != nil {
		t.Errorf("expected no error with nil database, got %v", err)
	}
}

func TestAuthPlugin_SetupEndpoints_WithDatabase(t *testing.T) {
	db := &mockDatabase{}
	plugin := &AuthPlugin{
		jwtSecret: "test-secret",
		db:        db,
		jwtTTL:    3600,
	}

	app := fiber.New()
	err := plugin.SetupEndpoints(app)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}


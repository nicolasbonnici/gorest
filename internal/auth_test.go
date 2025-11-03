//go:build integration

package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func TestSetupAuth(t *testing.T) {
	cleanupTestDB(t)

	// Create a test user
	ctx := context.Background()
	testEmail := "test@example.com"
	testPassword := "testpass123"
	testFirstname := "Test"
	testLastname := "User"

	// Hash the password using bcrypt
	passwordHash, err := HashPassword(testPassword)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Insert user with hashed password
	var query string
	switch db.DriverName() {
	case "postgres":
		query = `INSERT INTO users (email, password, firstname, lastname) VALUES ($1, $2, $3, $4)`
	case "mysql", "sqlite":
		query = `INSERT INTO users (email, password, firstname, lastname) VALUES (?, ?, ?, ?)`
	}
	_, err = db.Exec(ctx, query, testEmail, passwordHash, testFirstname, testLastname)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	jwtSecret := "test-secret-key"
	jwtTTL := 900

	tests := []struct {
		name           string
		body           map[string]string
		expectedStatus int
		checkToken     bool
	}{
		{
			name: "successful login",
			body: map[string]string{
				"email":    testEmail,
				"password": testPassword,
			},
			expectedStatus: 200,
			checkToken:     true,
		},
		{
			name: "wrong password",
			body: map[string]string{
				"email":    testEmail,
				"password": "wrongpassword",
			},
			expectedStatus: 401,
			checkToken:     false,
		},
		{
			name: "non-existent user",
			body: map[string]string{
				"email":    "nonexistent@example.com",
				"password": "anypassword",
			},
			expectedStatus: 401,
			checkToken:     false,
		},
		{
			name: "missing email",
			body: map[string]string{
				"password": testPassword,
			},
			expectedStatus: 400,
			checkToken:     false,
		},
		{
			name: "missing password",
			body: map[string]string{
				"email": testEmail,
			},
			expectedStatus: 400,
			checkToken:     false,
		},
		{
			name:           "invalid JSON body",
			body:           nil,
			expectedStatus: 400,
			checkToken:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new app for each subtest to reset rate limiters
			app := fiber.New()
			SetupAuth(app, db, jwtSecret, jwtTTL)

			var bodyBytes []byte
			if tt.body != nil {
				bodyBytes, _ = json.Marshal(tt.body)
			} else {
				bodyBytes = []byte("invalid json")
			}

			req := httptest.NewRequest("POST", "/login", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.checkToken {
				var result map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&result)

				token, ok := result["token"].(string)
				if !ok || token == "" {
					t.Error("Expected token in response")
				}

				// Verify JWT token
				parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
					return []byte(jwtSecret), nil
				})
				if err != nil || !parsedToken.Valid {
					t.Errorf("Invalid JWT token: %v", err)
				}

				// Verify user data in response
				user, ok := result["user"].(map[string]interface{})
				if !ok {
					t.Error("Expected user object in response")
				} else {
					if user["email"] != testEmail {
						t.Errorf("Expected email %s, got %s", testEmail, user["email"])
					}
					if user["firstname"] != testFirstname {
						t.Errorf("Expected firstname %s, got %s", testFirstname, user["firstname"])
					}
					if user["lastname"] != testLastname {
						t.Errorf("Expected lastname %s, got %s", testLastname, user["lastname"])
					}
				}
			}
		})
	}
}

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "basic hash",
			password: "testpass123",
		},
		{
			name:     "different password",
			password: "anotherpass456",
		},
		{
			name:     "empty password",
			password: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1, err := HashPassword(tt.password)
			if err != nil {
				t.Fatalf("Failed to hash password: %v", err)
			}

			// Verify hash is not empty
			if hash1 == "" {
				t.Error("Hash should not be empty")
			}

			// Verify hash starts with bcrypt prefix
			if len(hash1) < 4 || hash1[:4] != "$2a$" && hash1[:4] != "$2b$" && hash1[:4] != "$2y$" {
				t.Errorf("Hash should start with bcrypt prefix, got: %s", hash1[:4])
			}

			// Verify password verification works
			if err := verifyPassword(tt.password, hash1); err != nil {
				t.Errorf("Password verification failed: %v", err)
			}

			// Verify wrong password fails
			if err := verifyPassword("wrongpassword", hash1); err == nil {
				t.Error("Wrong password should not verify successfully")
			}

			// Verify bcrypt generates different salts (hashes are different each time)
			hash2, err := HashPassword(tt.password)
			if err != nil {
				t.Fatalf("Failed to hash password second time: %v", err)
			}
			if hash1 == hash2 {
				t.Error("Bcrypt should generate different salts, producing different hashes")
			}

			// But both hashes should verify the same password
			if err := verifyPassword(tt.password, hash2); err != nil {
				t.Errorf("Second hash verification failed: %v", err)
			}
		})
	}
}

func TestJWTTokenExpiration(t *testing.T) {
	cleanupTestDB(t)

	ctx := context.Background()
	testEmail := "jwttest@example.com"
	testPassword := "testpass123"

	passwordHash, err := HashPassword(testPassword)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	var query string
	switch db.DriverName() {
	case "postgres":
		query = `INSERT INTO users (email, password, firstname, lastname) VALUES ($1, $2, $3, $4)`
	case "mysql", "sqlite":
		query = `INSERT INTO users (email, password, firstname, lastname) VALUES (?, ?, ?, ?)`
	}
	_, err = db.Exec(ctx, query, testEmail, passwordHash, "JWT", "Test")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name         string
		jwtTTL       int
		validateFunc func(*testing.T, string, string, int)
	}{
		{
			name:   "token contains exp and iat claims",
			jwtTTL: 900,
			validateFunc: func(t *testing.T, token, secret string, ttl int) {
				parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
					return []byte(secret), nil
				})
				if err != nil {
					t.Fatalf("Failed to parse token: %v", err)
				}

				claims, ok := parsedToken.Claims.(jwt.MapClaims)
				if !ok {
					t.Fatal("Failed to extract claims")
				}

				iat, ok := claims["iat"].(float64)
				if !ok {
					t.Error("Token missing iat claim")
				}

				exp, ok := claims["exp"].(float64)
				if !ok {
					t.Error("Token missing exp claim")
				}

				expectedExpDiff := float64(ttl)
				actualExpDiff := exp - iat
				if actualExpDiff != expectedExpDiff {
					t.Errorf("Expected exp-iat difference of %v, got %v", expectedExpDiff, actualExpDiff)
				}

				if time.Unix(int64(exp), 0).Before(time.Now()) {
					t.Error("Token already expired")
				}
			},
		},
		{
			name:   "short TTL - 1 second",
			jwtTTL: 1,
			validateFunc: func(t *testing.T, token, secret string, ttl int) {
				time.Sleep(2 * time.Second)

				parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
					return []byte(secret), nil
				})

				if err == nil && parsedToken.Valid {
					t.Error("Token should be expired after TTL")
				}
			},
		},
		{
			name:   "valid token within TTL",
			jwtTTL: 3600,
			validateFunc: func(t *testing.T, token, secret string, ttl int) {
				parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
					return []byte(secret), nil
				})

				if err != nil {
					t.Fatalf("Token parsing failed: %v", err)
				}

				if !parsedToken.Valid {
					t.Error("Token should be valid within TTL")
				}

				claims, ok := parsedToken.Claims.(jwt.MapClaims)
				if !ok {
					t.Fatal("Failed to extract claims")
				}

				exp, ok := claims["exp"].(float64)
				if !ok {
					t.Fatal("Missing exp claim")
				}

				expiresAt := time.Unix(int64(exp), 0)
				if time.Now().After(expiresAt) {
					t.Error("Token expired prematurely")
				}
			},
		},
		{
			name:   "default TTL - 15 minutes",
			jwtTTL: 900,
			validateFunc: func(t *testing.T, token, secret string, ttl int) {
				parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
					return []byte(secret), nil
				})

				if err != nil {
					t.Fatalf("Token parsing failed: %v", err)
				}

				claims, ok := parsedToken.Claims.(jwt.MapClaims)
				if !ok {
					t.Fatal("Failed to extract claims")
				}

				exp, ok := claims["exp"].(float64)
				if !ok {
					t.Fatal("Missing exp claim")
				}

				iat, ok := claims["iat"].(float64)
				if !ok {
					t.Fatal("Missing iat claim")
				}

				actualTTL := int64(exp - iat)
				if actualTTL != int64(ttl) {
					t.Errorf("Expected TTL of %d seconds, got %d", ttl, actualTTL)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			jwtSecret := "test-secret-for-expiration"
			SetupAuth(app, db, jwtSecret, tt.jwtTTL)

			body := map[string]string{
				"email":    testEmail,
				"password": testPassword,
			}
			bodyBytes, _ := json.Marshal(body)

			req := httptest.NewRequest("POST", "/login", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test request: %v", err)
			}

			if resp.StatusCode != 200 {
				t.Fatalf("Login failed with status %d", resp.StatusCode)
			}

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)

			token, ok := result["token"].(string)
			if !ok || token == "" {
				t.Fatal("No token in response")
			}

			tt.validateFunc(t, token, jwtSecret, tt.jwtTTL)
		})
	}
}

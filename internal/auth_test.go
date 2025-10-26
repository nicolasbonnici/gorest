//go:build integration

package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

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

	// Insert user and get the generated ID
	var testUserId string
	var query string
	switch db.DriverName() {
	case "postgres":
		query = `INSERT INTO users (email, password, firstname, lastname) VALUES ($1, $2, $3, $4) RETURNING id`
		row := db.QueryRow(ctx, query, testEmail, hashPassword(testPassword, "temp"), testFirstname, testLastname)
		if err := row.Scan(&testUserId); err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
	case "mysql", "sqlite":
		// For MySQL/SQLite, insert without ID and let it auto-generate
		query = `INSERT INTO users (email, password, firstname, lastname) VALUES (?, ?, ?, ?)`
		_, err := db.Exec(ctx, query, testEmail, hashPassword(testPassword, "temp"), testFirstname, testLastname)
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
		// Query to get the ID
		selectQuery := "SELECT id FROM users WHERE email = ?"
		if db.DriverName() == "postgres" {
			selectQuery = "SELECT id FROM users WHERE email = $1"
		}
		row := db.QueryRow(ctx, selectQuery, testEmail)
		if err := row.Scan(&testUserId); err != nil {
			t.Fatalf("Failed to get user ID: %v", err)
		}
	}

	// Update password with correct hash using the actual user ID
	passwordHash := hashPassword(testPassword, testUserId)
	var updateQuery string
	switch db.DriverName() {
	case "postgres":
		updateQuery = `UPDATE users SET password = $1 WHERE id = $2`
	case "mysql", "sqlite":
		updateQuery = `UPDATE users SET password = ? WHERE id = ?`
	}
	_, err := db.Exec(ctx, updateQuery, passwordHash, testUserId)
	if err != nil {
		t.Fatalf("Failed to update user password: %v", err)
	}

	// Create Fiber app and setup auth
	app := fiber.New()
	jwtSecret := "test-secret-key"
	SetupAuth(app, db, jwtSecret)

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
		userId   string
		expected string
	}{
		{
			name:     "basic hash",
			password: "testpass123",
			userId:   "user-1",
			expected: hashPassword("testpass123", "user-1"),
		},
		{
			name:     "different user same password",
			password: "testpass123",
			userId:   "user-2",
			expected: hashPassword("testpass123", "user-2"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hashPassword(tt.password, tt.userId)

			if result != tt.expected {
				t.Errorf("Expected hash %s, got %s", tt.expected, result)
			}

			// Verify hash is deterministic
			result2 := hashPassword(tt.password, tt.userId)
			if result != result2 {
				t.Error("Hash should be deterministic")
			}

			// Verify different inputs produce different hashes
			if tt.userId == "user-1" {
				differentUser := hashPassword(tt.password, "user-2")
				if result == differentUser {
					t.Error("Different users should produce different hashes")
				}

				differentPassword := hashPassword("differentpass", tt.userId)
				if result == differentPassword {
					t.Error("Different passwords should produce different hashes")
				}
			}

			// Verify hash is hex-encoded SHA256 (64 characters)
			if len(result) != 64 {
				t.Errorf("Expected hash length 64, got %d", len(result))
			}
		})
	}
}

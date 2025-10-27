package helpers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func TestRequireAuth(t *testing.T) {
	app := fiber.New()
	jwtSecret := "test-secret-key"

	// Create a protected endpoint
	app.Get("/protected", RequireAuth(jwtSecret, func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	}))

	tests := []struct {
		name           string
		setupAuth      func() string
		expectedStatus int
		expectedError  string
	}{
		{
			name: "valid token",
			setupAuth: func() string {
				claims := jwt.MapClaims{"user_id": "123"}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenStr, _ := token.SignedString([]byte(jwtSecret))
				return "Bearer " + tokenStr
			},
			expectedStatus: 200,
		},
		{
			name: "missing authorization header",
			setupAuth: func() string {
				return ""
			},
			expectedStatus: 401,
			expectedError:  "missing token",
		},
		{
			name: "missing Bearer prefix",
			setupAuth: func() string {
				return "InvalidFormat token"
			},
			expectedStatus: 401,
			expectedError:  "missing token",
		},
		{
			name: "invalid token format",
			setupAuth: func() string {
				return "Bearer invalid.token.format"
			},
			expectedStatus: 401,
			expectedError:  "invalid token",
		},
		{
			name: "token signed with wrong secret",
			setupAuth: func() string {
				claims := jwt.MapClaims{"user_id": "123"}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenStr, _ := token.SignedString([]byte("wrong-secret"))
				return "Bearer " + tokenStr
			},
			expectedStatus: 401,
			expectedError:  "invalid token",
		},
		{
			name: "empty Bearer token",
			setupAuth: func() string {
				return "Bearer "
			},
			expectedStatus: 401,
			expectedError:  "missing token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/protected", nil)
			authHeader := tt.setupAuth()
			if authHeader != "" {
				req.Header.Set("Authorization", authHeader)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedError != "" {
				var result map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				errorMsg, ok := result["error"].(string)
				if !ok {
					t.Error("Expected error field in response")
				}
				if errorMsg != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, errorMsg)
				}
			}

			if tt.expectedStatus == 200 {
				var result map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				message, ok := result["message"].(string)
				if !ok || message != "success" {
					t.Error("Expected success message in response")
				}
			}
		})
	}
}

func TestRequireAuthWithDifferentClaims(t *testing.T) {
	app := fiber.New()
	jwtSecret := "test-secret-key"

	app.Get("/protected", RequireAuth(jwtSecret, func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	}))

	tests := []struct {
		name   string
		claims jwt.MapClaims
	}{
		{
			name: "with user_id claim",
			claims: jwt.MapClaims{
				"user_id": "123",
			},
		},
		{
			name: "with email claim",
			claims: jwt.MapClaims{
				"email": "test@example.com",
			},
		},
		{
			name: "with multiple claims",
			claims: jwt.MapClaims{
				"user_id":   "123",
				"email":     "test@example.com",
				"firstname": "Test",
				"lastname":  "User",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, tt.claims)
			tokenStr, err := token.SignedString([]byte(jwtSecret))
			if err != nil {
				t.Fatalf("Failed to sign token: %v", err)
			}

			req := httptest.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+tokenStr)

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test request: %v", err)
			}

			if resp.StatusCode != 200 {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		})
	}
}

//go:build integration

package health

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestSetupHealthCheck(t *testing.T) {
	app := fiber.New()
	SetupHealthCheck(app, db)

	tests := []struct {
		name           string
		expectedStatus int
		checkResponse  bool
	}{
		{
			name:           "health check returns healthy",
			expectedStatus: 200,
			checkResponse:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/health", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.checkResponse {
				var result map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				status, ok := result["status"].(string)
				if !ok {
					t.Error("Expected status field in response")
				}
				if status != "healthy" {
					t.Errorf("Expected status 'healthy', got %s", status)
				}

				database, ok := result["database"].(map[string]interface{})
				if !ok {
					t.Error("Expected database field in response")
				}

				dbStatus, ok := database["status"].(string)
				if !ok {
					t.Error("Expected database.status field in response")
				}
				if dbStatus != "up" {
					t.Errorf("Expected database status 'up', got %s", dbStatus)
				}
			}
		})
	}
}

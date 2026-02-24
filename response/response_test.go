package response

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestSetCommonHeaders(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		SetCommonHeaders(c)
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	poweredBy := resp.Header.Get("X-Powered-By")
	if poweredBy == "" {
		t.Error("Expected X-Powered-By header to be set")
	}
	// Just verify it starts with "GoREST/" since version may vary
	if len(poweredBy) < 7 || poweredBy[:7] != "GoREST/" {
		t.Errorf("Expected X-Powered-By to start with 'GoREST/', got '%s'", poweredBy)
	}
}

func TestSetContentTypeHeader(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{"JSON-LD format", "jsonld", "application/ld+json"},
		{"JSON format", "json", "application/json"},
		{"Other format defaults to JSON", "other", "application/json"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				SetContentTypeHeader(c, tt.format)
				return c.SendStatus(200)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, _ := app.Test(req)

			contentType := resp.Header.Get("Content-Type")
			if contentType != tt.expected {
				t.Errorf("Expected Content-Type '%s', got '%s'", tt.expected, contentType)
			}
		})
	}
}

func TestSendJSON(t *testing.T) {
	tests := []struct {
		name         string
		data         interface{}
		status       int
		acceptHeader string
	}{
		{
			name:         "Send JSON with default accept",
			data:         fiber.Map{"message": "test"},
			status:       200,
			acceptHeader: "",
		},
		{
			name:         "Send JSON with JSON accept",
			data:         fiber.Map{"message": "test"},
			status:       201,
			acceptHeader: "application/json",
		},
		{
			name:         "Send JSON with JSON-LD accept",
			data:         fiber.Map{"message": "test"},
			status:       200,
			acceptHeader: "application/ld+json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/test", func(c *fiber.Ctx) error {
				return SendJSON(c, tt.status, tt.data)
			})

			req := httptest.NewRequest("POST", "/test", nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}
			resp, _ := app.Test(req)

			if resp.StatusCode != tt.status {
				t.Errorf("Expected status %d, got %d", tt.status, resp.StatusCode)
			}

			// Fiber's .JSON() always sets Content-Type to application/json
			contentType := resp.Header.Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
			}

			poweredBy := resp.Header.Get("X-Powered-By")
			if poweredBy == "" {
				t.Error("Expected X-Powered-By header to be set")
			}
		})
	}
}

func TestParseExpandQuery(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected []string
	}{
		{
			name:     "No expand parameter",
			url:      "/test",
			expected: []string{},
		},
		{
			name:     "Single expand parameter",
			url:      "/test?expand[]=author",
			expected: []string{"author"},
		},
		{
			name:     "Multiple expand parameters",
			url:      "/test?expand[]=author&expand[]=comments",
			expected: []string{"author", "comments"},
		},
		{
			name:     "Expand with other parameters",
			url:      "/test?page=1&expand[]=author&limit=10",
			expected: []string{"author"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			var result []string

			app.Get("/test", func(c *fiber.Ctx) error {
				result = ParseExpandQuery(c)
				return c.SendStatus(200)
			})

			req := httptest.NewRequest("GET", tt.url, nil)
			_, err := app.Test(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d expand values, got %d", len(tt.expected), len(result))
				return
			}

			for i, exp := range tt.expected {
				if i >= len(result) || result[i] != exp {
					t.Errorf("Expected expand[%d]='%s', got '%s'", i, exp, result[i])
				}
			}
		})
	}
}

func TestParseAcceptHeader(t *testing.T) {
	tests := []struct {
		name     string
		accept   string
		expected []string
	}{
		{
			name:     "Single content type",
			accept:   "application/json",
			expected: []string{"application/json"},
		},
		{
			name:     "Multiple content types",
			accept:   "application/json,application/ld+json",
			expected: []string{"application/json", "application/ld+json"},
		},
		{
			name:     "With quality values",
			accept:   "application/json;q=0.9,text/html;q=0.8",
			expected: []string{"application/json", "text/html"},
		},
		{
			name:     "Empty header",
			accept:   "",
			expected: []string{},
		},
		{
			name:     "With spaces",
			accept:   "application/json, application/ld+json",
			expected: []string{"application/json", "application/ld+json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAcceptHeader(tt.accept)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d content types, got %d", len(tt.expected), len(result))
				return
			}
			for i, ct := range result {
				if ct != tt.expected[i] {
					t.Errorf("Expected content type %s at index %d, got %s", tt.expected[i], i, ct)
				}
			}
		})
	}
}

func TestDetermineFormat(t *testing.T) {
	tests := []struct {
		name           string
		acceptHeader   string
		expectedFormat string
	}{
		{
			name:           "JSON-LD preferred",
			acceptHeader:   "application/ld+json",
			expectedFormat: "jsonld",
		},
		{
			name:           "JSON only",
			acceptHeader:   "application/json",
			expectedFormat: "json",
		},
		{
			name:           "JSON then JSON-LD",
			acceptHeader:   "application/json,application/ld+json",
			expectedFormat: "json",
		},
		{
			name:           "JSON-LD then JSON",
			acceptHeader:   "application/ld+json,application/json",
			expectedFormat: "jsonld",
		},
		{
			name:           "No accept header defaults to JSON-LD",
			acceptHeader:   "",
			expectedFormat: "jsonld",
		},
		{
			name:           "Other content type defaults to JSON-LD",
			acceptHeader:   "text/html",
			expectedFormat: "jsonld",
		},
		{
			name:           "JSON with quality values",
			acceptHeader:   "application/json;q=0.9",
			expectedFormat: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				format := DetermineFormat(c)
				return c.SendString(format)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test request: %v", err)
			}

			var result string
			buf := make([]byte, 100)
			n, _ := resp.Body.Read(buf)
			result = string(buf[:n])

			if result != tt.expectedFormat {
				t.Errorf("Expected format %s, got %s", tt.expectedFormat, result)
			}
		})
	}
}

func TestSendFormatted(t *testing.T) {
	testData := map[string]interface{}{
		"id":    "123",
		"title": "Test Todo",
	}

	tests := []struct {
		name                string
		acceptHeader        string
		expectedStatus      int
		checkContentType    bool
		expectedContentType string
	}{
		{
			name:                "Send JSON-LD formatted response",
			acceptHeader:        "application/ld+json",
			expectedStatus:      200,
			checkContentType:    true,
			expectedContentType: "application/ld+json",
		},
		{
			name:                "Send JSON formatted response",
			acceptHeader:        "application/json",
			expectedStatus:      200,
			checkContentType:    true,
			expectedContentType: "application/json",
		},
		{
			name:                "Send with 201 status",
			acceptHeader:        "application/json",
			expectedStatus:      201,
			checkContentType:    true,
			expectedContentType: "application/json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				return SendFormatted(c, tt.expectedStatus, testData)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Accept", tt.acceptHeader)

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.checkContentType {
				contentType := resp.Header.Get("Content-Type")
				if contentType != tt.expectedContentType {
					t.Errorf("Expected Content-Type %s, got %s", tt.expectedContentType, contentType)
				}
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Errorf("Failed to decode response: %v", err)
			}
		})
	}
}

func TestSendFormattedWithArray(t *testing.T) {
	app := fiber.New()

	testData := []map[string]interface{}{
		{"id": "1", "title": "Todo 1"},
		{"id": "2", "title": "Todo 2"},
	}

	app.Get("/test", func(c *fiber.Ctx) error {
		return SendFormatted(c, 200, testData)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test request: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}
}

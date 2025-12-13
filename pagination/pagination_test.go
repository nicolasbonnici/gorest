package pagination

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/response"
)

func TestParseIntQuery(t *testing.T) {
	tests := []struct {
		name         string
		queryValue   string
		defaultValue int
		maxValue     int
		expected     int
	}{
		{"empty query uses default", "", 100, 1000, 100},
		{"valid value", "50", 100, 1000, 50},
		{"exceeds max returns max", "2000", 100, 1000, 1000},
		{"negative value uses default", "-10", 100, 1000, 100},
		{"invalid value uses default", "abc", 100, 1000, 100},
		{"zero value", "0", 100, 1000, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", func(c *fiber.Ctx) error {
				result := ParseIntQuery(c, "limit", tt.defaultValue, tt.maxValue)
				return c.SendString(string(rune(result)))
			})

			req := httptest.NewRequest("GET", "/test?limit="+tt.queryValue, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test: %v", err)
			}
			defer resp.Body.Close()

			app2 := fiber.New()
			app2.Get("/verify", func(c *fiber.Ctx) error {
				c.QueryParser(&struct {
					Limit string `query:"limit"`
				}{})
				result := ParseIntQuery(c, "limit", tt.defaultValue, tt.maxValue)
				if result != tt.expected {
					t.Errorf("Expected %d, got %d", tt.expected, result)
				}
				return c.SendStatus(200)
			})

			req2 := httptest.NewRequest("GET", "/verify?limit="+tt.queryValue, nil)
			_, _ = app2.Test(req2)
		})
	}
}

func TestSendHydraCollection(t *testing.T) {
	app := fiber.New()

	app.Get("/items", func(c *fiber.Ctx) error {
		items := []map[string]string{
			{"id": "1", "name": "Item 1"},
			{"id": "2", "name": "Item 2"},
		}
		total := 10
		return SendHydraCollection(c, items, &total, 2, 1, 2)
	})

	req := httptest.NewRequest("GET", "/items", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result HydraCollection
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Context != "http://www.w3.org/ns/hydra/context.jsonld" {
		t.Errorf("Wrong context: %s", result.Context)
	}

	if result.Type != "hydra:Collection" {
		t.Errorf("Wrong type: %s", result.Type)
	}

	if result.TotalItems == nil || *result.TotalItems != 10 {
		t.Errorf("Expected total 10, got %v", result.TotalItems)
	}

	if result.View == nil {
		t.Fatal("View should not be nil")
	}

	if result.View.Next == nil {
		t.Error("Next link should be present")
	}

	if result.View.Previous != nil {
		t.Error("Previous link should not be present on first page")
	}
}

func TestSendHydraCollectionWithoutCount(t *testing.T) {
	app := fiber.New()

	app.Get("/items", func(c *fiber.Ctx) error {
		items := []string{"item1", "item2"}
		return SendHydraCollection(c, items, nil, 2, 1, 2)
	})

	req := httptest.NewRequest("GET", "/items", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}
	defer resp.Body.Close()

	var result HydraCollection
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.TotalItems != nil {
		t.Error("TotalItems should be nil when count not requested")
	}

	if result.View.Last != nil {
		t.Error("Last link should not be present without total count")
	}
}

func TestSendHydraCollectionPagination(t *testing.T) {
	app := fiber.New()

	app.Get("/items", func(c *fiber.Ctx) error {
		items := []int{3, 4}
		total := 10
		return SendHydraCollection(c, items, &total, 2, 2, 2)
	})

	req := httptest.NewRequest("GET", "/items?limit=2&page=2", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}
	defer resp.Body.Close()

	var result HydraCollection
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.View.Previous == nil {
		t.Error("Previous link should be present on page 2")
	}

	if result.View.Next == nil {
		t.Error("Next link should be present")
	}
}

func TestSendPaginatedError(t *testing.T) {
	app := fiber.New()

	app.Get("/error", func(c *fiber.Ctx) error {
		return SendPaginatedError(c, 500, "Test error")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 500 {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["@type"] != "hydra:Error" {
		t.Errorf("Wrong error type: %v", result["@type"])
	}

	if result["hydra:description"] != "Test error" {
		t.Errorf("Wrong error message: %v", result["hydra:description"])
	}
}

func TestPaginationXPoweredByHeader(t *testing.T) {
	response.Initialize("test-version")

	tests := []struct {
		name    string
		handler fiber.Handler
	}{
		{
			name: "SendHydraCollection",
			handler: func(c *fiber.Ctx) error {
				items := []string{"item1"}
				total := 1
				return SendHydraCollection(c, items, &total, 1, 1, 1)
			},
		},
		{
			name: "SendPaginatedError",
			handler: func(c *fiber.Ctx) error {
				return SendPaginatedError(c, 400, "test error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/test", tt.handler)

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to test: %v", err)
			}
			defer resp.Body.Close()

			poweredBy := resp.Header.Get("X-Powered-By")
			if poweredBy != "GoREST/test-version" {
				t.Errorf("Expected X-Powered-By header to be 'GoREST/test-version', got '%s'", poweredBy)
			}
		})
	}
}

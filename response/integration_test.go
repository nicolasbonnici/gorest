package response

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestResponseXPoweredByHeader(t *testing.T) {
	// Initialize response package with version
	Initialize("1.2.3-test")

	app := fiber.New()

	// Test SendFormatted
	app.Get("/formatted", func(c *fiber.Ctx) error {
		return SendFormatted(c, 200, map[string]string{"test": "data"})
	})

	// Test SendError
	app.Get("/error", func(c *fiber.Ctx) error {
		return SendError(c, 400, "test error")
	})

	// Test SendSuccess
	app.Get("/success", func(c *fiber.Ctx) error {
		return SendSuccess(c, map[string]string{"test": "success"})
	})

	// Test SendCreated
	app.Get("/created", func(c *fiber.Ctx) error {
		return SendCreated(c, map[string]string{"test": "created"})
	})

	endpoints := []string{"/formatted", "/error", "/success", "/created"}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req := httptest.NewRequest("GET", endpoint, nil)
			resp, _ := app.Test(req)

			header := resp.Header.Get("X-Powered-By")
			expected := "GoREST/1.2.3-test"
			if header != expected {
				t.Errorf("Expected X-Powered-By '%s', got '%s'", expected, header)
			}
		})
	}
}

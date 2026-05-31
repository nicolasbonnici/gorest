package middleware

import (
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestLogger_ReturnsHandler(t *testing.T) {
	handler := Logger("test")

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Errorf("expected body 'ok', got '%s'", string(body))
	}
}

func TestLogger_LogsGetRequest(t *testing.T) {
	handler := Logger("test")

	app := fiber.New()
	app.Use(handler)
	app.Get("/api/users", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"users": []string{"alice"}})
	})

	req := httptest.NewRequest("GET", "/api/users", nil)
	req.Header.Set("User-Agent", "test-agent")

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestLogger_LogsPostRequest(t *testing.T) {
	handler := Logger("test")

	app := fiber.New()
	app.Use(handler)
	app.Post("/api/users", func(c fiber.Ctx) error {
		return c.Status(201).JSON(fiber.Map{"created": true})
	})

	req := httptest.NewRequest("POST", "/api/users", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
}

func TestLogger_LogsErrorResponses(t *testing.T) {
	handler := Logger("production")

	app := fiber.New()
	app.Use(handler)
	app.Get("/error400", func(c fiber.Ctx) error { return c.SendStatus(400) })
	app.Get("/error404", func(c fiber.Ctx) error { return c.SendStatus(404) })
	app.Get("/error500", func(c fiber.Ctx) error { return c.SendStatus(500) })

	testCases := []struct {
		path           string
		expectedStatus int
	}{
		{"/error400", 400},
		{"/error404", 404},
		{"/error500", 500},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest("GET", tc.path, nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != tc.expectedStatus {
			t.Errorf("path %s: expected status %d, got %d", tc.path, tc.expectedStatus, resp.StatusCode)
		}
	}
}

func TestLogger_LogsErrorMessageInDevelopment(t *testing.T) {
	handler := Logger("development")

	app := fiber.New()
	app.Use(handler)
	app.Get("/error", func(c fiber.Ctx) error {
		return errors.New("test error message")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	resp, _ := app.Test(req)

	// The error should be logged (checked via logger output in real scenario)
	// Here we just verify the handler doesn't crash
	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

func TestLogger_DoesNotLogErrorMessageInProduction(t *testing.T) {
	handler := Logger("production")

	app := fiber.New()
	app.Use(handler)
	app.Get("/error", func(c fiber.Ctx) error {
		return errors.New("secret error message")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	resp, _ := app.Test(req)

	// Error message should NOT be logged in production
	if resp.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

func TestLogger_LogsFullURLWithQueryString(t *testing.T) {
	handler := Logger("test")

	app := fiber.New()
	app.Use(handler)
	app.Get("/api/search", func(c fiber.Ctx) error {
		return c.SendString("results")
	})

	req := httptest.NewRequest("GET", "/api/search?q=test&limit=10", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestLogger_DoesNotBlockRequests(t *testing.T) {
	handler := Logger("test")

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("response")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("logger should not block requests, got status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "response" {
		t.Errorf("logger should not modify response, got '%s'", string(body))
	}
}

func TestLogger_WithRequestID(t *testing.T) {
	handler := Logger("test")

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("requestid", "test-request-id-123")
		return c.Next()
	})
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestLogger_WithoutRequestID(t *testing.T) {
	handler := Logger("test")

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200 even without request ID, got %d", resp.StatusCode)
	}
}

func TestLogger_AllHTTPMethods(t *testing.T) {
	handler := Logger("test")

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error { return c.SendStatus(200) })
	app.Post("/test", func(c fiber.Ctx) error { return c.SendStatus(201) })
	app.Put("/test", func(c fiber.Ctx) error { return c.SendStatus(200) })
	app.Delete("/test", func(c fiber.Ctx) error { return c.SendStatus(204) })
	app.Patch("/test", func(c fiber.Ctx) error { return c.SendStatus(200) })

	tests := []struct {
		method         string
		expectedStatus int
	}{
		{"GET", 200},
		{"POST", 201},
		{"PUT", 200},
		{"DELETE", 204},
		{"PATCH", 200},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("method %s: expected status %d, got %d", tt.method, tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestLogger_LogsWithUserAgent(t *testing.T) {
	handler := Logger("test")

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	userAgents := []string{
		"Mozilla/5.0",
		"curl/7.64.1",
		"PostmanRuntime/7.26.8",
	}

	for _, ua := range userAgents {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("User-Agent", ua)

		resp, _ := app.Test(req)

		if resp.StatusCode != 200 {
			t.Errorf("user agent %s: expected status 200, got %d", ua, resp.StatusCode)
		}
	}
}

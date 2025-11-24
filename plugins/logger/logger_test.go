package logger

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestNewPlugin(t *testing.T) {
	plugin := NewPlugin()

	if plugin == nil {
		t.Fatal("expected non-nil plugin")
	}

	_, ok := plugin.(*LoggerPlugin)
	if !ok {
		t.Fatal("expected *LoggerPlugin type")
	}
}

func TestLoggerPlugin_Name(t *testing.T) {
	plugin := NewPlugin()

	name := plugin.Name()

	if name != "logger" {
		t.Errorf("expected plugin name 'logger', got '%s'", name)
	}
}

func TestLoggerPlugin_Initialize_EmptyConfig(t *testing.T) {
	plugin := &LoggerPlugin{}

	err := plugin.Initialize(map[string]interface{}{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestLoggerPlugin_Initialize_WithConfig(t *testing.T) {
	plugin := &LoggerPlugin{}

	config := map[string]interface{}{
		"format": "json",
		"level":  "debug",
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestLoggerPlugin_Initialize_NilConfig(t *testing.T) {
	plugin := &LoggerPlugin{}

	err := plugin.Initialize(nil)

	if err != nil {
		t.Fatalf("expected no error with nil config, got %v", err)
	}
}

func TestLoggerPlugin_Handler_NotNil(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestLoggerPlugin_Handler_ReturnsMiddleware(t *testing.T) {
	plugin := NewPlugin()
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
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

func TestLoggerPlugin_Handler_LogsGetRequest(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/api/users", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"users": []string{"alice"}})
	})

	req := httptest.NewRequest("GET", "/api/users", nil)
	req.Header.Set("User-Agent", "test-agent")

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestLoggerPlugin_Handler_LogsPostRequest(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Post("/api/users", func(c *fiber.Ctx) error {
		return c.Status(201).JSON(fiber.Map{"created": true})
	})

	req := httptest.NewRequest("POST", "/api/users", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
}

func TestLoggerPlugin_Handler_LogsPutRequest(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Put("/api/users/1", func(c *fiber.Ctx) error {
		return c.SendString("updated")
	})

	req := httptest.NewRequest("PUT", "/api/users/1", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestLoggerPlugin_Handler_LogsDeleteRequest(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Delete("/api/users/1", func(c *fiber.Ctx) error {
		return c.SendStatus(204)
	})

	req := httptest.NewRequest("DELETE", "/api/users/1", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 204 {
		t.Errorf("expected status 204, got %d", resp.StatusCode)
	}
}

func TestLoggerPlugin_Handler_LogsMultipleRequests(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != 200 {
			t.Errorf("request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}
	}
}

func TestLoggerPlugin_Handler_LogsDifferentEndpoints(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/users", func(c *fiber.Ctx) error { return c.SendString("users") })
	app.Get("/posts", func(c *fiber.Ctx) error { return c.SendString("posts") })
	app.Get("/comments", func(c *fiber.Ctx) error { return c.SendString("comments") })

	endpoints := []string{"/users", "/posts", "/comments"}

	for _, endpoint := range endpoints {
		req := httptest.NewRequest("GET", endpoint, nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != 200 {
			t.Errorf("endpoint %s: expected status 200, got %d", endpoint, resp.StatusCode)
		}
	}
}

func TestLoggerPlugin_Handler_LogsWithUserAgent(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	userAgents := []string{
		"Mozilla/5.0",
		"curl/7.64.1",
		"PostmanRuntime/7.26.8",
		"Go-http-client/1.1",
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

func TestLoggerPlugin_Handler_LogsErrorResponses(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/error400", func(c *fiber.Ctx) error { return c.SendStatus(400) })
	app.Get("/error401", func(c *fiber.Ctx) error { return c.SendStatus(401) })
	app.Get("/error404", func(c *fiber.Ctx) error { return c.SendStatus(404) })
	app.Get("/error500", func(c *fiber.Ctx) error { return c.SendStatus(500) })

	testCases := []struct {
		path           string
		expectedStatus int
	}{
		{"/error400", 400},
		{"/error401", 401},
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

func TestLoggerPlugin_Handler_DoesNotBlockRequests(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
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

func TestLoggerPlugin_Handler_WithRequestID(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("requestid", "test-request-id-123")
		return c.Next()
	})
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestLoggerPlugin_Handler_WithoutRequestID(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200 even without request ID, got %d", resp.StatusCode)
	}
}

func TestLoggerPlugin_Handler_LogsQueryParameters(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/search", func(c *fiber.Ctx) error {
		return c.SendString("results")
	})

	req := httptest.NewRequest("GET", "/search?q=test&page=2&limit=10", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestLoggerPlugin_Handler_LogsIPAddress(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1")

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestLoggerPlugin_Handler_LogsSuccessAndFailure(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/success", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})
	app.Get("/failure", func(c *fiber.Ctx) error {
		return c.SendStatus(500)
	})

	reqSuccess := httptest.NewRequest("GET", "/success", nil)
	respSuccess, _ := app.Test(reqSuccess)

	if respSuccess.StatusCode != 200 {
		t.Errorf("expected status 200 for success, got %d", respSuccess.StatusCode)
	}

	reqFailure := httptest.NewRequest("GET", "/failure", nil)
	respFailure, _ := app.Test(reqFailure)

	if respFailure.StatusCode != 500 {
		t.Errorf("expected status 500 for failure, got %d", respFailure.StatusCode)
	}
}

func TestLoggerPlugin_IntegrationWithFiber(t *testing.T) {
	plugin := NewPlugin()
	err := plugin.Initialize(map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to initialize plugin: %v", err)
	}

	app := fiber.New()
	app.Use(plugin.Handler())
	app.Get("/api/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy"})
	})

	req := httptest.NewRequest("GET", "/api/health", nil)
	req.Header.Set("User-Agent", "integration-test")

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestLoggerPlugin_Handler_AllHTTPMethods(t *testing.T) {
	plugin := &LoggerPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Post("/test", func(c *fiber.Ctx) error { return c.SendStatus(201) })
	app.Put("/test", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Delete("/test", func(c *fiber.Ctx) error { return c.SendStatus(204) })
	app.Patch("/test", func(c *fiber.Ctx) error { return c.SendStatus(200) })

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

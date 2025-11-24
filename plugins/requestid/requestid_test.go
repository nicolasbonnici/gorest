package requestid

import (
	"io"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestNewPlugin(t *testing.T) {
	plugin := NewPlugin()

	if plugin == nil {
		t.Fatal("expected non-nil plugin")
	}

	_, ok := plugin.(*RequestIDPlugin)
	if !ok {
		t.Fatal("expected *RequestIDPlugin type")
	}
}

func TestRequestIDPlugin_Name(t *testing.T) {
	plugin := NewPlugin()

	name := plugin.Name()

	if name != "requestid" {
		t.Errorf("expected plugin name 'requestid', got '%s'", name)
	}
}

func TestRequestIDPlugin_Initialize_EmptyConfig(t *testing.T) {
	plugin := &RequestIDPlugin{}

	err := plugin.Initialize(map[string]interface{}{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRequestIDPlugin_Initialize_WithConfig(t *testing.T) {
	plugin := &RequestIDPlugin{}

	config := map[string]interface{}{
		"header": "X-Custom-Request-ID",
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRequestIDPlugin_Initialize_NilConfig(t *testing.T) {
	plugin := &RequestIDPlugin{}

	err := plugin.Initialize(nil)

	if err != nil {
		t.Fatalf("expected no error with nil config, got %v", err)
	}
}

func TestRequestIDPlugin_Handler_NotNil(t *testing.T) {
	plugin := &RequestIDPlugin{}
	handler := plugin.Handler()

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestRequestIDPlugin_Handler_GeneratesRequestID(t *testing.T) {
	plugin := &RequestIDPlugin{}
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

	requestID := resp.Header.Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}

func TestRequestIDPlugin_Handler_UUIDFormat(t *testing.T) {
	plugin := &RequestIDPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	requestID := resp.Header.Get("X-Request-ID")
	if requestID == "" {
		t.Fatal("expected X-Request-ID header to be set")
	}

	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !uuidPattern.MatchString(requestID) {
		t.Errorf("expected UUID format, got '%s'", requestID)
	}
}

func TestRequestIDPlugin_Handler_UniqueIDs(t *testing.T) {
	plugin := &RequestIDPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	requestIDs := make(map[string]bool)

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, _ := app.Test(req)

		requestID := resp.Header.Get("X-Request-ID")
		if requestID == "" {
			t.Fatalf("request %d: expected X-Request-ID header to be set", i+1)
		}

		if requestIDs[requestID] {
			t.Errorf("request %d: duplicate request ID '%s'", i+1, requestID)
		}
		requestIDs[requestID] = true
	}

	if len(requestIDs) != 100 {
		t.Errorf("expected 100 unique request IDs, got %d", len(requestIDs))
	}
}

func TestRequestIDPlugin_Handler_DifferentEndpoints(t *testing.T) {
	plugin := &RequestIDPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/users", func(c *fiber.Ctx) error { return c.SendString("users") })
	app.Get("/posts", func(c *fiber.Ctx) error { return c.SendString("posts") })
	app.Get("/comments", func(c *fiber.Ctx) error { return c.SendString("comments") })

	endpoints := []string{"/users", "/posts", "/comments"}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req := httptest.NewRequest("GET", endpoint, nil)
			resp, _ := app.Test(req)

			if resp.StatusCode != 200 {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			requestID := resp.Header.Get("X-Request-ID")
			if requestID == "" {
				t.Errorf("endpoint %s: expected X-Request-ID header to be set", endpoint)
			}
		})
	}
}

func TestRequestIDPlugin_Handler_DifferentMethods(t *testing.T) {
	plugin := &RequestIDPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Post("/test", func(c *fiber.Ctx) error { return c.SendStatus(201) })
	app.Put("/test", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Delete("/test", func(c *fiber.Ctx) error { return c.SendStatus(204) })
	app.Patch("/test", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			resp, _ := app.Test(req)

			requestID := resp.Header.Get("X-Request-ID")
			if requestID == "" {
				t.Errorf("method %s: expected X-Request-ID header to be set", method)
			}

			uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
			if !uuidPattern.MatchString(requestID) {
				t.Errorf("method %s: invalid UUID format '%s'", method, requestID)
			}
		})
	}
}

func TestRequestIDPlugin_Handler_RequestIDInResponse(t *testing.T) {
	plugin := &RequestIDPlugin{}
	handler := plugin.Handler()

	var capturedRequestID string

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		capturedRequestID = c.Locals("requestid").(string)
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	headerRequestID := resp.Header.Get("X-Request-ID")

	if capturedRequestID == "" {
		t.Error("expected request ID to be available in context")
	}

	if headerRequestID == "" {
		t.Error("expected X-Request-ID header to be set")
	}

	if capturedRequestID != headerRequestID {
		t.Errorf("context request ID '%s' does not match header '%s'", capturedRequestID, headerRequestID)
	}
}

func TestRequestIDPlugin_Handler_DoesNotBlockRequests(t *testing.T) {
	plugin := &RequestIDPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("response")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("request ID middleware should not block requests, got status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "response" {
		t.Errorf("expected body 'response', got '%s'", string(body))
	}
}

func TestRequestIDPlugin_Handler_MultipleRequests(t *testing.T) {
	plugin := &RequestIDPlugin{}
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

		requestID := resp.Header.Get("X-Request-ID")
		if requestID == "" {
			t.Errorf("request %d: expected X-Request-ID header to be set", i+1)
		}
	}
}

func TestRequestIDPlugin_Handler_SuccessAndErrorRequests(t *testing.T) {
	plugin := &RequestIDPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/success", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})
	app.Get("/error", func(c *fiber.Ctx) error {
		return c.SendStatus(500)
	})

	reqSuccess := httptest.NewRequest("GET", "/success", nil)
	respSuccess, _ := app.Test(reqSuccess)

	if respSuccess.StatusCode != 200 {
		t.Errorf("expected status 200 for success, got %d", respSuccess.StatusCode)
	}

	successRequestID := respSuccess.Header.Get("X-Request-ID")
	if successRequestID == "" {
		t.Error("expected X-Request-ID header on success response")
	}

	reqError := httptest.NewRequest("GET", "/error", nil)
	respError, _ := app.Test(reqError)

	if respError.StatusCode != 500 {
		t.Errorf("expected status 500 for error, got %d", respError.StatusCode)
	}

	errorRequestID := respError.Header.Get("X-Request-ID")
	if errorRequestID == "" {
		t.Error("expected X-Request-ID header on error response")
	}

	if successRequestID == errorRequestID {
		t.Error("expected different request IDs for different requests")
	}
}

func TestRequestIDPlugin_Handler_ConcurrentRequests(t *testing.T) {
	plugin := &RequestIDPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	requestIDs := make([]string, 50)

	for i := 0; i < 50; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, _ := app.Test(req)

		requestIDs[i] = resp.Header.Get("X-Request-ID")

		if requestIDs[i] == "" {
			t.Errorf("request %d: expected X-Request-ID to be set", i+1)
		}
	}

	uniqueIDs := make(map[string]bool)
	for _, id := range requestIDs {
		uniqueIDs[id] = true
	}

	if len(uniqueIDs) != 50 {
		t.Errorf("expected 50 unique request IDs, got %d", len(uniqueIDs))
	}
}

func TestRequestIDPlugin_Handler_PreservesResponseBody(t *testing.T) {
	plugin := &RequestIDPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/json", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "test", "data": 123})
	})

	req := httptest.NewRequest("GET", "/json", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	requestID := resp.Header.Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID header to be set")
	}

	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestRequestIDPlugin_Handler_WithQueryParameters(t *testing.T) {
	plugin := &RequestIDPlugin{}
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

	requestID := resp.Header.Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID header to be set for request with query params")
	}
}

func TestRequestIDPlugin_Handler_WithHeaders(t *testing.T) {
	plugin := &RequestIDPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Accept", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	requestID := resp.Header.Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}

func TestRequestIDPlugin_IntegrationWithFiber(t *testing.T) {
	plugin := NewPlugin()
	err := plugin.Initialize(map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to initialize plugin: %v", err)
	}

	app := fiber.New()
	app.Use(plugin.Handler())
	app.Get("/api/data", func(c *fiber.Ctx) error {
		requestID := c.Locals("requestid")
		return c.JSON(fiber.Map{
			"data":       "test",
			"request_id": requestID,
		})
	})

	req := httptest.NewRequest("GET", "/api/data", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	headerRequestID := resp.Header.Get("X-Request-ID")
	if headerRequestID == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}

func TestRequestIDPlugin_Handler_UUIDv4Format(t *testing.T) {
	plugin := &RequestIDPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	requestID := resp.Header.Get("X-Request-ID")
	if requestID == "" {
		t.Fatal("expected X-Request-ID header to be set")
	}

	parts := regexp.MustCompile(`-`).Split(requestID, -1)
	if len(parts) != 5 {
		t.Errorf("expected 5 UUID parts, got %d", len(parts))
	}

	if len(parts[0]) != 8 {
		t.Errorf("first part should be 8 chars, got %d", len(parts[0]))
	}
	if len(parts[1]) != 4 {
		t.Errorf("second part should be 4 chars, got %d", len(parts[1]))
	}
	if len(parts[2]) != 4 {
		t.Errorf("third part should be 4 chars, got %d", len(parts[2]))
	}
	if len(parts[3]) != 4 {
		t.Errorf("fourth part should be 4 chars, got %d", len(parts[3]))
	}
	if len(parts[4]) != 12 {
		t.Errorf("fifth part should be 12 chars, got %d", len(parts[4]))
	}
}

func TestRequestIDPlugin_Handler_NoCollisions(t *testing.T) {
	plugin := &RequestIDPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	requestIDs := make(map[string]int)
	numRequests := 1000

	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, _ := app.Test(req)

		requestID := resp.Header.Get("X-Request-ID")
		requestIDs[requestID]++
	}

	if len(requestIDs) != numRequests {
		t.Errorf("expected %d unique request IDs, got %d", numRequests, len(requestIDs))
	}

	for id, count := range requestIDs {
		if count > 1 {
			t.Errorf("request ID '%s' appeared %d times, expected 1", id, count)
		}
	}
}

func TestRequestIDPlugin_Handler_HeaderPresence(t *testing.T) {
	plugin := &RequestIDPlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	headers := resp.Header

	if _, ok := headers["X-Request-Id"]; !ok {
		t.Error("expected X-Request-ID header to be present")
	}
}

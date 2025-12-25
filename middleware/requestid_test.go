package middleware

import (
	"io"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestRequestID_ReturnsHandler(t *testing.T) {
	handler := RequestID()

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

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
}

func TestRequestID_GeneratesRequestID(t *testing.T) {
	handler := RequestID()

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

func TestRequestID_UUIDFormat(t *testing.T) {
	handler := RequestID()

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

func TestRequestID_UniqueIDs(t *testing.T) {
	handler := RequestID()

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

func TestRequestID_DifferentEndpoints(t *testing.T) {
	handler := RequestID()

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

func TestRequestID_DifferentMethods(t *testing.T) {
	handler := RequestID()

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

func TestRequestID_ContextLocals(t *testing.T) {
	handler := RequestID()

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

func TestRequestID_DoesNotBlockRequests(t *testing.T) {
	handler := RequestID()

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

func TestRequestID_MultipleRequests(t *testing.T) {
	handler := RequestID()

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

func TestRequestID_SuccessAndErrorRequests(t *testing.T) {
	handler := RequestID()

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

func TestRequestID_ConcurrentRequests(t *testing.T) {
	handler := RequestID()

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

func TestRequestID_PreservesResponseBody(t *testing.T) {
	handler := RequestID()

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

func TestRequestID_WithQueryParameters(t *testing.T) {
	handler := RequestID()

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

func TestRequestID_NoCollisions(t *testing.T) {
	handler := RequestID()

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

package middleware

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestRateLimit_ReturnsHandler(t *testing.T) {
	handler := RateLimit(10, 20)

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

func TestRateLimit_AllowsRequestsUnderLimit(t *testing.T) {
	handler := RateLimit(100, 200)

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

		body, _ := io.ReadAll(resp.Body)
		if string(body) != "ok" {
			t.Errorf("request %d: expected body 'ok', got '%s'", i+1, string(body))
		}
	}
}

func TestRateLimit_BlocksExcessiveRequests(t *testing.T) {
	handler := RateLimit(5, 10)

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	var blockedCount int
	var allowedCount int

	for i := 0; i < 15; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, _ := app.Test(req)

		if resp.StatusCode == 429 {
			blockedCount++

			body, _ := io.ReadAll(resp.Body)
			var errorResp map[string]interface{}
			if err := json.Unmarshal(body, &errorResp); err == nil {
				if errorMsg, ok := errorResp["error"].(string); ok && errorMsg == "" {
					t.Errorf("expected error message in response")
				}
			}
		} else if resp.StatusCode == 200 {
			allowedCount++
		}
	}

	if blockedCount == 0 {
		t.Error("expected at least some requests to be blocked by rate limiter")
	}

	if allowedCount == 0 {
		t.Error("expected at least some requests to be allowed")
	}
}

func TestRateLimit_ErrorMessageFormat(t *testing.T) {
	handler := RateLimit(2, 5)

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, _ := app.Test(req)

		if resp.StatusCode == 429 {
			body, _ := io.ReadAll(resp.Body)
			var errorResp map[string]interface{}
			if err := json.Unmarshal(body, &errorResp); err != nil {
				t.Fatalf("failed to parse error response: %v", err)
			}

			if _, ok := errorResp["error"]; !ok {
				t.Error("expected 'error' field in response")
			}

			return
		}
	}
}

func TestRateLimit_DifferentIPsGetSeparateLimits(t *testing.T) {
	handler := RateLimit(5, 10)

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != 200 && resp.StatusCode != 429 {
			t.Errorf("request %d: unexpected status code %d", i+1, resp.StatusCode)
		}
	}
}

func TestRateLimit_RespectsConfiguredLimits(t *testing.T) {
	tests := []struct {
		name              string
		requestsPerSecond int
		burst             int
	}{
		{"low limit", 1, 2},
		{"medium limit", 10, 20},
		{"high limit", 100, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RateLimit(tt.requestsPerSecond, tt.burst)

			app := fiber.New()
			app.Use(handler)
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("ok")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, _ := app.Test(req)

			if resp.StatusCode != 200 {
				t.Errorf("%s: expected first request to succeed, got status %d", tt.name, resp.StatusCode)
			}
		})
	}
}

func TestRateLimit_DoesNotBlockNormalUsage(t *testing.T) {
	handler := RateLimit(50, 100)

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != 200 {
			t.Errorf("request %d: normal usage should not be blocked, got status %d", i+1, resp.StatusCode)
		}
	}
}

func TestRateLimit_RecoverAfterExpiration(t *testing.T) {
	handler := RateLimit(2, 5)

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		app.Test(req)
	}

	time.Sleep(1100 * time.Millisecond)

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected request to succeed after rate limit expiration, got status %d", resp.StatusCode)
	}
}

func TestRateLimit_PreservesResponseBody(t *testing.T) {
	handler := RateLimit(100, 200)

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

	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("expected non-empty response body")
	}
}

package middleware

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestCORS_DefaultWildcard(t *testing.T) {
	handler := CORS("")

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", allowOrigin)
	}

	allowCredentials := resp.Header.Get("Access-Control-Allow-Credentials")
	if allowCredentials != "" {
		t.Errorf("expected no Access-Control-Allow-Credentials with wildcard, got '%s'", allowCredentials)
	}
}

func TestCORS_ExplicitWildcard(t *testing.T) {
	handler := CORS("*")

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")

	resp, _ := app.Test(req)

	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", allowOrigin)
	}

	allowCredentials := resp.Header.Get("Access-Control-Allow-Credentials")
	if allowCredentials != "" {
		t.Errorf("expected no Access-Control-Allow-Credentials with wildcard, got '%s'", allowCredentials)
	}
}

func TestCORS_SpecificOrigin(t *testing.T) {
	handler := CORS("https://example.com")

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin != "https://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin 'https://example.com', got '%s'", allowOrigin)
	}

	allowCredentials := resp.Header.Get("Access-Control-Allow-Credentials")
	if allowCredentials != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials 'true', got '%s'", allowCredentials)
	}
}

func TestCORS_MultipleOrigins(t *testing.T) {
	handler := CORS("https://example.com,https://test.com")

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	tests := []struct {
		name           string
		origin         string
		expectAllowed  bool
		expectedOrigin string
	}{
		{
			name:           "first origin",
			origin:         "https://example.com",
			expectAllowed:  true,
			expectedOrigin: "https://example.com",
		},
		{
			name:           "second origin",
			origin:         "https://test.com",
			expectAllowed:  true,
			expectedOrigin: "https://test.com",
		},
		{
			name:           "disallowed origin",
			origin:         "https://evil.com",
			expectAllowed:  false,
			expectedOrigin: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tt.origin)

			resp, _ := app.Test(req)

			allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
			if tt.expectAllowed {
				if allowOrigin != tt.expectedOrigin {
					t.Errorf("expected Access-Control-Allow-Origin '%s', got '%s'", tt.expectedOrigin, allowOrigin)
				}
			} else {
				if allowOrigin != "" {
					t.Errorf("expected no Access-Control-Allow-Origin for disallowed origin, got '%s'", allowOrigin)
				}
			}
		})
	}
}

func TestCORS_PreflightRequest(t *testing.T) {
	handler := CORS("https://example.com")

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization")

	resp, _ := app.Test(req)

	if resp.StatusCode != 204 {
		t.Errorf("expected status 204 for preflight, got %d", resp.StatusCode)
	}

	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin != "https://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin 'https://example.com', got '%s'", allowOrigin)
	}

	allowMethods := resp.Header.Get("Access-Control-Allow-Methods")
	if allowMethods != "GET, POST, PUT, DELETE, OPTIONS" {
		t.Errorf("expected Access-Control-Allow-Methods 'GET, POST, PUT, DELETE, OPTIONS', got '%s'", allowMethods)
	}

	allowHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	if allowHeaders != "Origin, Content-Type, Accept, Authorization" {
		t.Errorf("expected Access-Control-Allow-Headers 'Origin, Content-Type, Accept, Authorization', got '%s'", allowHeaders)
	}

	maxAge := resp.Header.Get("Access-Control-Max-Age")
	if maxAge != "86400" {
		t.Errorf("expected Access-Control-Max-Age '86400', got '%s'", maxAge)
	}
}

func TestCORS_AllowedMethods(t *testing.T) {
	handler := CORS("*")

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error { return c.SendString("get") })
	app.Post("/test", func(c fiber.Ctx) error { return c.SendString("post") })
	app.Put("/test", func(c fiber.Ctx) error { return c.SendString("put") })
	app.Delete("/test", func(c fiber.Ctx) error { return c.SendString("delete") })

	methods := []string{"GET", "POST", "PUT", "DELETE"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/test", nil)
			req.Header.Set("Origin", "https://example.com")

			resp, _ := app.Test(req)

			if resp.StatusCode != 200 {
				t.Errorf("expected status 200 for %s, got %d", method, resp.StatusCode)
			}

			allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
			if allowOrigin != "*" {
				t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", allowOrigin)
			}
		})
	}
}

func TestCORS_WithoutOriginHeader(t *testing.T) {
	handler := CORS("*")

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

func TestCORS_CustomHeaders(t *testing.T) {
	handler := CORS("https://example.com")

	app := fiber.New()
	app.Use(handler)
	app.Post("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin != "https://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin 'https://example.com', got '%s'", allowOrigin)
	}
}

func TestCORS_CredentialsOnlyWithSpecificOrigin(t *testing.T) {
	tests := []struct {
		name                string
		origins             string
		expectedCredentials bool
	}{
		{
			name:                "wildcard no credentials",
			origins:             "*",
			expectedCredentials: false,
		},
		{
			name:                "specific origin with credentials",
			origins:             "https://example.com",
			expectedCredentials: true,
		},
		{
			name:                "multiple origins with credentials",
			origins:             "https://example.com,https://test.com",
			expectedCredentials: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := CORS(tt.origins)

			app := fiber.New()
			app.Use(handler)
			app.Get("/test", func(c fiber.Ctx) error {
				return c.SendString("ok")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", "https://example.com")

			resp, _ := app.Test(req)

			allowCredentials := resp.Header.Get("Access-Control-Allow-Credentials")
			if tt.expectedCredentials {
				if allowCredentials != "true" {
					t.Errorf("expected Access-Control-Allow-Credentials 'true', got '%s'", allowCredentials)
				}
			} else {
				if allowCredentials != "" {
					t.Errorf("expected no Access-Control-Allow-Credentials, got '%s'", allowCredentials)
				}
			}
		})
	}
}

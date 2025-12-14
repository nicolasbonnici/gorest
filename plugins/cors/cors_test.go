package cors

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

	corsPlugin, ok := plugin.(*CORSPlugin)
	if !ok {
		t.Fatal("expected *CORSPlugin type")
	}

	if corsPlugin.origins != "*" {
		t.Errorf("expected default origins '*', got '%s'", corsPlugin.origins)
	}
}

func TestCORSPlugin_Name(t *testing.T) {
	plugin := NewPlugin()

	name := plugin.Name()

	if name != "cors" {
		t.Errorf("expected plugin name 'cors', got '%s'", name)
	}
}

func TestCORSPlugin_Initialize_DefaultConfig(t *testing.T) {
	plugin := &CORSPlugin{}

	err := plugin.Initialize(map[string]interface{}{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.origins != "" {
		t.Errorf("expected empty origins after empty init, got '%s'", plugin.origins)
	}
}

func TestCORSPlugin_Initialize_WithStringOrigins(t *testing.T) {
	plugin := &CORSPlugin{}

	config := map[string]interface{}{
		"origins": "https://example.com",
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.origins != "https://example.com" {
		t.Errorf("expected origins 'https://example.com', got '%s'", plugin.origins)
	}
}

func TestCORSPlugin_Initialize_WithMultipleOrigins(t *testing.T) {
	plugin := &CORSPlugin{}

	config := map[string]interface{}{
		"origins": "https://example.com,https://test.com,https://api.example.com",
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := "https://example.com,https://test.com,https://api.example.com"
	if plugin.origins != expected {
		t.Errorf("expected origins '%s', got '%s'", expected, plugin.origins)
	}
}

func TestCORSPlugin_Initialize_WithWildcard(t *testing.T) {
	plugin := &CORSPlugin{}

	config := map[string]interface{}{
		"origins": "*",
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.origins != "*" {
		t.Errorf("expected origins '*', got '%s'", plugin.origins)
	}
}

func TestCORSPlugin_Initialize_NonStringOrigins(t *testing.T) {
	plugin := &CORSPlugin{origins: "initial"}

	config := map[string]interface{}{
		"origins": 12345,
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.origins != "initial" {
		t.Errorf("expected origins to remain 'initial', got '%s'", plugin.origins)
	}
}

func TestCORSPlugin_Handler_DefaultWildcard(t *testing.T) {
	plugin := &CORSPlugin{origins: "*"}
	handler := plugin.Handler()

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
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

func TestCORSPlugin_Handler_SpecificOrigin(t *testing.T) {
	plugin := &CORSPlugin{origins: "https://example.com"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
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

func TestCORSPlugin_Handler_MultipleOrigins(t *testing.T) {
	plugin := &CORSPlugin{origins: "https://example.com,https://test.com"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
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

func TestCORSPlugin_Handler_PreflightRequest(t *testing.T) {
	plugin := &CORSPlugin{origins: "https://example.com"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
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
	if allowMethods != "GET,POST,PUT,DELETE,OPTIONS" {
		t.Errorf("expected Access-Control-Allow-Methods 'GET,POST,PUT,DELETE,OPTIONS', got '%s'", allowMethods)
	}

	allowHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	if allowHeaders != "Origin,Content-Type,Accept,Authorization" {
		t.Errorf("expected Access-Control-Allow-Headers 'Origin,Content-Type,Accept,Authorization', got '%s'", allowHeaders)
	}

	maxAge := resp.Header.Get("Access-Control-Max-Age")
	if maxAge != "86400" {
		t.Errorf("expected Access-Control-Max-Age '86400', got '%s'", maxAge)
	}
}

func TestCORSPlugin_Handler_AllowedMethods(t *testing.T) {
	plugin := &CORSPlugin{origins: "*"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendString("get") })
	app.Post("/test", func(c *fiber.Ctx) error { return c.SendString("post") })
	app.Put("/test", func(c *fiber.Ctx) error { return c.SendString("put") })
	app.Delete("/test", func(c *fiber.Ctx) error { return c.SendString("delete") })

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

func TestCORSPlugin_Handler_WithoutOriginHeader(t *testing.T) {
	plugin := &CORSPlugin{origins: "*"}
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

func TestCORSPlugin_Handler_CustomHeaders(t *testing.T) {
	plugin := &CORSPlugin{origins: "https://example.com"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Post("/test", func(c *fiber.Ctx) error {
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

func TestCORSPlugin_Handler_CredentialsOnlyWithSpecificOrigin(t *testing.T) {
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
			plugin := &CORSPlugin{origins: tt.origins}
			handler := plugin.Handler()

			app := fiber.New()
			app.Use(handler)
			app.Get("/test", func(c *fiber.Ctx) error {
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

func TestCORSPlugin_Handler_ComplexPreflightScenario(t *testing.T) {
	plugin := &CORSPlugin{origins: "https://app.example.com,https://admin.example.com"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Post("/api/data", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"data": "success"})
	})

	req := httptest.NewRequest("OPTIONS", "/api/data", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type,Authorization,X-Custom-Header")

	resp, _ := app.Test(req)

	if resp.StatusCode != 204 {
		t.Errorf("expected status 204, got %d", resp.StatusCode)
	}

	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin != "https://app.example.com" {
		t.Errorf("expected Access-Control-Allow-Origin 'https://app.example.com', got '%s'", allowOrigin)
	}

	allowCredentials := resp.Header.Get("Access-Control-Allow-Credentials")
	if allowCredentials != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials 'true', got '%s'", allowCredentials)
	}

	allowMethods := resp.Header.Get("Access-Control-Allow-Methods")
	if allowMethods == "" {
		t.Error("expected Access-Control-Allow-Methods to be set")
	}

	allowHeaders := resp.Header.Get("Access-Control-Allow-Headers")
	if allowHeaders == "" {
		t.Error("expected Access-Control-Allow-Headers to be set")
	}
}

func TestCORSPlugin_IntegrationWithFiber(t *testing.T) {
	plugin := NewPlugin()
	err := plugin.Initialize(map[string]interface{}{
		"origins": "https://example.com",
	})
	if err != nil {
		t.Fatalf("failed to initialize plugin: %v", err)
	}

	app := fiber.New()
	app.Use(plugin.Handler())
	app.Get("/api/users", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"users": []string{"alice", "bob"}})
	})

	req := httptest.NewRequest("GET", "/api/users", nil)
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

package security

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestNewPlugin(t *testing.T) {
	plugin := NewPlugin()

	if plugin == nil {
		t.Fatal("expected non-nil plugin")
	}

	secPlugin, ok := plugin.(*SecurityPlugin)
	if !ok {
		t.Fatal("expected *SecurityPlugin type")
	}

	if secPlugin.version != "dev" {
		t.Errorf("expected default version 'dev', got '%s'", secPlugin.version)
	}
}

func TestSecurityPlugin_Name(t *testing.T) {
	plugin := NewPlugin()

	name := plugin.Name()

	if name != "security" {
		t.Errorf("expected plugin name 'security', got '%s'", name)
	}
}

func TestSecurityPlugin_Initialize_EmptyConfig(t *testing.T) {
	plugin := &SecurityPlugin{version: "initial"}

	err := plugin.Initialize(map[string]interface{}{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.version != "initial" {
		t.Errorf("expected version to remain 'initial', got '%s'", plugin.version)
	}
}

func TestSecurityPlugin_Initialize_WithVersion(t *testing.T) {
	plugin := &SecurityPlugin{}

	config := map[string]interface{}{
		"__version": "1.2.3",
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.version != "1.2.3" {
		t.Errorf("expected version '1.2.3', got '%s'", plugin.version)
	}
}

func TestSecurityPlugin_Initialize_NonStringVersion(t *testing.T) {
	plugin := &SecurityPlugin{version: "default"}

	config := map[string]interface{}{
		"__version": 12345,
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.version != "default" {
		t.Errorf("expected version to remain 'default', got '%s'", plugin.version)
	}
}

func TestSecurityPlugin_Handler_AllHeaders(t *testing.T) {
	plugin := &SecurityPlugin{version: "1.0.0"}
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
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	expectedHeaders := map[string]string{
		"X-Powered-By":               "GoREST/1.0.0",
		"X-Content-Type-Options":     "nosniff",
		"X-Frame-Options":            "DENY",
		"X-XSS-Protection":           "1; mode=block",
		"Strict-Transport-Security":  "max-age=31536000; includeSubDomains",
		"Referrer-Policy":            "strict-origin-when-cross-origin",
		"Permissions-Policy":         "geolocation=(), microphone=(), camera=()",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := resp.Header.Get(header)
		if actualValue != expectedValue {
			t.Errorf("header %s: expected '%s', got '%s'", header, expectedValue, actualValue)
		}
	}
}

func TestSecurityPlugin_Handler_XPoweredBy(t *testing.T) {
	tests := []struct {
		name            string
		version         string
		expectedHeader  string
	}{
		{
			name:           "dev version",
			version:        "dev",
			expectedHeader: "GoREST/dev",
		},
		{
			name:           "release version",
			version:        "2.5.1",
			expectedHeader: "GoREST/2.5.1",
		},
		{
			name:           "custom version",
			version:        "custom-build",
			expectedHeader: "GoREST/custom-build",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := &SecurityPlugin{version: tt.version}
			handler := plugin.Handler()

			app := fiber.New()
			app.Use(handler)
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendString("ok")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, _ := app.Test(req)

			poweredBy := resp.Header.Get("X-Powered-By")
			if poweredBy != tt.expectedHeader {
				t.Errorf("expected X-Powered-By '%s', got '%s'", tt.expectedHeader, poweredBy)
			}
		})
	}
}

func TestSecurityPlugin_Handler_XContentTypeOptions(t *testing.T) {
	plugin := &SecurityPlugin{version: "1.0.0"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	header := resp.Header.Get("X-Content-Type-Options")
	if header != "nosniff" {
		t.Errorf("expected X-Content-Type-Options 'nosniff', got '%s'", header)
	}
}

func TestSecurityPlugin_Handler_XFrameOptions(t *testing.T) {
	plugin := &SecurityPlugin{version: "1.0.0"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	header := resp.Header.Get("X-Frame-Options")
	if header != "DENY" {
		t.Errorf("expected X-Frame-Options 'DENY', got '%s'", header)
	}
}

func TestSecurityPlugin_Handler_XXSSProtection(t *testing.T) {
	plugin := &SecurityPlugin{version: "1.0.0"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	header := resp.Header.Get("X-XSS-Protection")
	if header != "1; mode=block" {
		t.Errorf("expected X-XSS-Protection '1; mode=block', got '%s'", header)
	}
}

func TestSecurityPlugin_Handler_StrictTransportSecurity(t *testing.T) {
	plugin := &SecurityPlugin{version: "1.0.0"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	header := resp.Header.Get("Strict-Transport-Security")
	expected := "max-age=31536000; includeSubDomains"
	if header != expected {
		t.Errorf("expected Strict-Transport-Security '%s', got '%s'", expected, header)
	}
}

func TestSecurityPlugin_Handler_ReferrerPolicy(t *testing.T) {
	plugin := &SecurityPlugin{version: "1.0.0"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	header := resp.Header.Get("Referrer-Policy")
	if header != "strict-origin-when-cross-origin" {
		t.Errorf("expected Referrer-Policy 'strict-origin-when-cross-origin', got '%s'", header)
	}
}

func TestSecurityPlugin_Handler_PermissionsPolicy(t *testing.T) {
	plugin := &SecurityPlugin{version: "1.0.0"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	header := resp.Header.Get("Permissions-Policy")
	expected := "geolocation=(), microphone=(), camera=()"
	if header != expected {
		t.Errorf("expected Permissions-Policy '%s', got '%s'", expected, header)
	}
}

func TestSecurityPlugin_Handler_MultipleRequests(t *testing.T) {
	plugin := &SecurityPlugin{version: "1.0.0"}
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

		xContentType := resp.Header.Get("X-Content-Type-Options")
		if xContentType != "nosniff" {
			t.Errorf("request %d: security headers missing or incorrect", i+1)
		}
	}
}

func TestSecurityPlugin_Handler_DifferentMethods(t *testing.T) {
	plugin := &SecurityPlugin{version: "1.0.0"}
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
			resp, _ := app.Test(req)

			if resp.StatusCode != 200 {
				t.Errorf("expected status 200 for %s, got %d", method, resp.StatusCode)
			}

			xFrameOptions := resp.Header.Get("X-Frame-Options")
			if xFrameOptions != "DENY" {
				t.Errorf("%s: expected X-Frame-Options 'DENY', got '%s'", method, xFrameOptions)
			}

			hsts := resp.Header.Get("Strict-Transport-Security")
			if hsts == "" {
				t.Errorf("%s: expected Strict-Transport-Security header to be set", method)
			}
		})
	}
}

func TestSecurityPlugin_Handler_MultipleEndpoints(t *testing.T) {
	plugin := &SecurityPlugin{version: "2.0.0"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/api/users", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"users": []string{"alice"}})
	})
	app.Get("/api/posts", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"posts": []string{"post1"}})
	})
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("healthy")
	})

	endpoints := []string{"/api/users", "/api/posts", "/health"}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req := httptest.NewRequest("GET", endpoint, nil)
			resp, _ := app.Test(req)

			if resp.StatusCode != 200 {
				t.Errorf("expected status 200 for %s, got %d", endpoint, resp.StatusCode)
			}

			poweredBy := resp.Header.Get("X-Powered-By")
			if poweredBy != "GoREST/2.0.0" {
				t.Errorf("%s: expected X-Powered-By 'GoREST/2.0.0', got '%s'", endpoint, poweredBy)
			}

			xssProtection := resp.Header.Get("X-XSS-Protection")
			if xssProtection != "1; mode=block" {
				t.Errorf("%s: expected X-XSS-Protection '1; mode=block', got '%s'", endpoint, xssProtection)
			}
		})
	}
}

func TestSecurityPlugin_Handler_HeadersPresent(t *testing.T) {
	plugin := &SecurityPlugin{version: "1.0.0"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	requiredHeaders := []string{
		"X-Powered-By",
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-XSS-Protection",
		"Strict-Transport-Security",
		"Referrer-Policy",
		"Permissions-Policy",
	}

	for _, header := range requiredHeaders {
		value := resp.Header.Get(header)
		if value == "" {
			t.Errorf("expected header %s to be set, but it was empty", header)
		}
	}
}

func TestSecurityPlugin_Handler_NoContentSecurityPolicy(t *testing.T) {
	plugin := &SecurityPlugin{version: "1.0.0"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	csp := resp.Header.Get("Content-Security-Policy")
	if csp != "" {
		t.Errorf("did not expect Content-Security-Policy header, got '%s'", csp)
	}
}

func TestSecurityPlugin_IntegrationWithFiber(t *testing.T) {
	plugin := NewPlugin()

	config := map[string]interface{}{
		"__version": "3.1.4",
	}

	err := plugin.Initialize(config)
	if err != nil {
		t.Fatalf("failed to initialize plugin: %v", err)
	}

	app := fiber.New()
	app.Use(plugin.Handler())
	app.Get("/secure", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "secure"})
	})

	req := httptest.NewRequest("GET", "/secure", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	poweredBy := resp.Header.Get("X-Powered-By")
	if poweredBy != "GoREST/3.1.4" {
		t.Errorf("expected X-Powered-By 'GoREST/3.1.4', got '%s'", poweredBy)
	}

	xFrameOptions := resp.Header.Get("X-Frame-Options")
	if xFrameOptions != "DENY" {
		t.Errorf("expected X-Frame-Options 'DENY', got '%s'", xFrameOptions)
	}
}

func TestSecurityPlugin_Handler_DoesNotBlockRequests(t *testing.T) {
	plugin := &SecurityPlugin{version: "1.0.0"}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("content")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("security headers should not block valid requests, got status %d", resp.StatusCode)
	}
}

func TestSecurityPlugin_Handler_EmptyVersion(t *testing.T) {
	plugin := &SecurityPlugin{version: ""}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	poweredBy := resp.Header.Get("X-Powered-By")
	if poweredBy != "GoREST/" {
		t.Errorf("expected X-Powered-By 'GoREST/', got '%s'", poweredBy)
	}
}

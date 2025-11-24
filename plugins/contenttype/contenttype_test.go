package contenttype

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestNewPlugin(t *testing.T) {
	plugin := NewPlugin()

	if plugin == nil {
		t.Fatal("expected non-nil plugin")
	}

	_, ok := plugin.(*ContentTypePlugin)
	if !ok {
		t.Fatal("expected *ContentTypePlugin type")
	}
}

func TestContentTypePlugin_Name(t *testing.T) {
	plugin := NewPlugin()

	name := plugin.Name()

	if name != "contenttype" {
		t.Errorf("expected plugin name 'contenttype', got '%s'", name)
	}
}

func TestContentTypePlugin_Initialize_EmptyConfig(t *testing.T) {
	plugin := &ContentTypePlugin{}

	err := plugin.Initialize(map[string]interface{}{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestContentTypePlugin_Initialize_WithConfig(t *testing.T) {
	plugin := &ContentTypePlugin{}

	config := map[string]interface{}{
		"strict": true,
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestContentTypePlugin_Initialize_NilConfig(t *testing.T) {
	plugin := &ContentTypePlugin{}

	err := plugin.Initialize(nil)

	if err != nil {
		t.Fatalf("expected no error with nil config, got %v", err)
	}
}

func TestContentTypePlugin_Handler_NotNil(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestContentTypePlugin_Handler_PostWithJSON(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("created")
	})

	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "created" {
		t.Errorf("expected body 'created', got '%s'", string(body))
	}
}

func TestContentTypePlugin_Handler_PostWithoutContentType(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("created")
	})

	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"key":"value"}`))

	resp, _ := app.Test(req)

	if resp.StatusCode != 415 {
		t.Errorf("expected status 415, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["error"] != "Content-Type must be application/json" {
		t.Errorf("expected error 'Content-Type must be application/json', got '%v'", result["error"])
	}
}

func TestContentTypePlugin_Handler_PostWithWrongContentType(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("created")
	})

	req := httptest.NewRequest("POST", "/test", strings.NewReader(`key=value`))
	req.Header.Set("Content-Type", "text/plain")

	resp, _ := app.Test(req)

	if resp.StatusCode != 415 {
		t.Errorf("expected status 415, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if result["error"] != "Content-Type must be application/json" {
		t.Errorf("expected error 'Content-Type must be application/json', got '%v'", result["error"])
	}
}

func TestContentTypePlugin_Handler_PostWithFormData(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("created")
	})

	req := httptest.NewRequest("POST", "/test", strings.NewReader(`key=value`))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, _ := app.Test(req)

	if resp.StatusCode != 415 {
		t.Errorf("expected status 415, got %d", resp.StatusCode)
	}
}

func TestContentTypePlugin_Handler_PostWithJSONCharset(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("created")
	})

	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200 with charset, got %d", resp.StatusCode)
	}
}

func TestContentTypePlugin_Handler_PutWithJSON(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Put("/test", func(c *fiber.Ctx) error {
		return c.SendString("updated")
	})

	req := httptest.NewRequest("PUT", "/test", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestContentTypePlugin_Handler_PutWithoutContentType(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Put("/test", func(c *fiber.Ctx) error {
		return c.SendString("updated")
	})

	req := httptest.NewRequest("PUT", "/test", strings.NewReader(`{"key":"value"}`))

	resp, _ := app.Test(req)

	if resp.StatusCode != 415 {
		t.Errorf("expected status 415, got %d", resp.StatusCode)
	}
}

func TestContentTypePlugin_Handler_PatchWithJSON(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Patch("/test", func(c *fiber.Ctx) error {
		return c.SendString("patched")
	})

	req := httptest.NewRequest("PATCH", "/test", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestContentTypePlugin_Handler_PatchWithoutContentType(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Patch("/test", func(c *fiber.Ctx) error {
		return c.SendString("patched")
	})

	req := httptest.NewRequest("PATCH", "/test", strings.NewReader(`{"key":"value"}`))

	resp, _ := app.Test(req)

	if resp.StatusCode != 415 {
		t.Errorf("expected status 415, got %d", resp.StatusCode)
	}
}

func TestContentTypePlugin_Handler_GetWithoutContentType(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200 for GET without Content-Type, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Errorf("expected body 'ok', got '%s'", string(body))
	}
}

func TestContentTypePlugin_Handler_DeleteWithoutContentType(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Delete("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(204)
	})

	req := httptest.NewRequest("DELETE", "/test", nil)

	resp, _ := app.Test(req)

	if resp.StatusCode != 204 {
		t.Errorf("expected status 204 for DELETE without Content-Type, got %d", resp.StatusCode)
	}
}

func TestContentTypePlugin_Handler_OptionsWithoutContentType(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Options("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil)

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200 for OPTIONS without Content-Type, got %d", resp.StatusCode)
	}
}

func TestContentTypePlugin_Handler_HeadWithoutContentType(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("HEAD", "/test", nil)

	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200 for HEAD without Content-Type, got %d", resp.StatusCode)
	}
}

func TestContentTypePlugin_Handler_AllMethodsValidation(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Post("/test", func(c *fiber.Ctx) error { return c.SendStatus(201) })
	app.Put("/test", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Patch("/test", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Delete("/test", func(c *fiber.Ctx) error { return c.SendStatus(204) })
	app.Options("/test", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	tests := []struct {
		method               string
		contentType          string
		shouldPass           bool
		expectedStatus       int
	}{
		{"GET", "", true, 200},
		{"POST", "application/json", true, 201},
		{"POST", "", false, 415},
		{"POST", "text/plain", false, 415},
		{"PUT", "application/json", true, 200},
		{"PUT", "", false, 415},
		{"PATCH", "application/json", true, 200},
		{"PATCH", "", false, 415},
		{"DELETE", "", true, 204},
		{"OPTIONS", "", true, 200},
	}

	for _, tt := range tests {
		t.Run(tt.method+"_"+tt.contentType, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("method %s with Content-Type '%s': expected status %d, got %d",
					tt.method, tt.contentType, tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestContentTypePlugin_Handler_MultiplePostRequests(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"created": true})
	})

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"id":`+string(rune(i))+`}`))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)

		if resp.StatusCode != 200 {
			t.Errorf("request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}
	}
}

func TestContentTypePlugin_Handler_MultipleEndpoints(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Post("/users", func(c *fiber.Ctx) error { return c.SendStatus(201) })
	app.Post("/posts", func(c *fiber.Ctx) error { return c.SendStatus(201) })
	app.Put("/comments/1", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	endpoints := []struct {
		method string
		path   string
	}{
		{"POST", "/users"},
		{"POST", "/posts"},
		{"PUT", "/comments/1"},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, strings.NewReader(`{"data":"value"}`))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)

		if resp.StatusCode != 200 && resp.StatusCode != 201 {
			t.Errorf("%s %s: expected status 200/201, got %d", ep.method, ep.path, resp.StatusCode)
		}
	}
}

func TestContentTypePlugin_Handler_JSONWithDifferentCharsets(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	charsets := []string{
		"application/json",
		"application/json; charset=utf-8",
		"application/json; charset=UTF-8",
		"application/json;charset=utf-8",
		"application/json; charset=iso-8859-1",
	}

	for _, ct := range charsets {
		t.Run(ct, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"key":"value"}`))
			req.Header.Set("Content-Type", ct)

			resp, _ := app.Test(req)

			if resp.StatusCode != 200 {
				t.Errorf("Content-Type '%s': expected status 200, got %d", ct, resp.StatusCode)
			}
		})
	}
}

func TestContentTypePlugin_Handler_InvalidContentTypes(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	invalidTypes := []string{
		"text/html",
		"text/xml",
		"application/xml",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
		"text/plain",
		"application/octet-stream",
	}

	for _, ct := range invalidTypes {
		t.Run(ct, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", strings.NewReader(`data`))
			req.Header.Set("Content-Type", ct)

			resp, _ := app.Test(req)

			if resp.StatusCode != 415 {
				t.Errorf("Content-Type '%s': expected status 415, got %d", ct, resp.StatusCode)
			}
		})
	}
}

func TestContentTypePlugin_Handler_CaseSensitivity(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	tests := []struct {
		contentType    string
		expectedStatus int
	}{
		{"application/json", 200},
		{"Application/JSON", 415},
		{"APPLICATION/JSON", 415},
		{"application/JSON", 415},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"key":"value"}`))
			req.Header.Set("Content-Type", tt.contentType)

			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Content-Type '%s': expected status %d, got %d", tt.contentType, tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestContentTypePlugin_IntegrationWithFiber(t *testing.T) {
	plugin := NewPlugin()
	err := plugin.Initialize(map[string]interface{}{})
	if err != nil {
		t.Fatalf("failed to initialize plugin: %v", err)
	}

	app := fiber.New()
	app.Use(plugin.Handler())
	app.Get("/api/users", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"users": []string{"alice", "bob"}})
	})
	app.Post("/api/users", func(c *fiber.Ctx) error {
		return c.Status(201).JSON(fiber.Map{"created": true})
	})

	getReq := httptest.NewRequest("GET", "/api/users", nil)
	getResp, _ := app.Test(getReq)

	if getResp.StatusCode != 200 {
		t.Errorf("GET: expected status 200, got %d", getResp.StatusCode)
	}

	postReq := httptest.NewRequest("POST", "/api/users", strings.NewReader(`{"name":"charlie"}`))
	postReq.Header.Set("Content-Type", "application/json")
	postResp, _ := app.Test(postReq)

	if postResp.StatusCode != 201 {
		t.Errorf("POST: expected status 201, got %d", postResp.StatusCode)
	}
}

func TestContentTypePlugin_Handler_EmptyContentType(t *testing.T) {
	plugin := &ContentTypePlugin{}
	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Post("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"key":"value"}`))
	req.Header.Set("Content-Type", "")

	resp, _ := app.Test(req)

	if resp.StatusCode != 415 {
		t.Errorf("expected status 415 for empty Content-Type, got %d", resp.StatusCode)
	}
}

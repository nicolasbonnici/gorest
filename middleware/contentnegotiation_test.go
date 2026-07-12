package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestContentNegotiation_ReturnsHandler(t *testing.T) {
	handler := ContentNegotiation()

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
}

func TestContentNegotiation_AllowsGETRequests(t *testing.T) {
	handler := ContentNegotiation()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("GET requests should not require Content-Type, got status %d", resp.StatusCode)
	}
}

func TestContentNegotiation_RequiresJSONForPOST(t *testing.T) {
	handler := ContentNegotiation()

	app := fiber.New()
	app.Use(handler)
	app.Post("/test", func(c fiber.Ctx) error {
		return c.SendStatus(201)
	})

	t.Run("without content-type", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		resp, _ := app.Test(req)

		if resp.StatusCode != 415 {
			t.Errorf("expected status 415, got %d", resp.StatusCode)
		}

		body, _ := io.ReadAll(resp.Body)
		var errorResp map[string]interface{}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			if _, ok := errorResp["error"]; !ok {
				t.Error("expected error field in response")
			}
		}
	})

	t.Run("with application/json", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)

		if resp.StatusCode != 201 {
			t.Errorf("expected status 201, got %d", resp.StatusCode)
		}
	})

	t.Run("with application/json; charset=utf-8", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		resp, _ := app.Test(req)

		if resp.StatusCode != 201 {
			t.Errorf("expected status 201 with charset, got %d", resp.StatusCode)
		}
	})

	t.Run("with text/plain", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", strings.NewReader("data"))
		req.Header.Set("Content-Type", "text/plain")
		resp, _ := app.Test(req)

		if resp.StatusCode != 415 {
			t.Errorf("expected status 415 for non-JSON content, got %d", resp.StatusCode)
		}
	})

	t.Run("with multipart/form-data allows file uploads", func(t *testing.T) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		fw, _ := w.CreateFormFile("file", "f.txt")
		_, _ = fw.Write([]byte("data"))
		_ = w.Close()

		req := httptest.NewRequest("POST", "/test", &buf)
		req.Header.Set("Content-Type", w.FormDataContentType())
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode == 415 {
			t.Error("multipart/form-data must be accepted for file uploads")
		}
	})
}

func TestContentNegotiation_RequiresJSONForPUT(t *testing.T) {
	handler := ContentNegotiation()

	app := fiber.New()
	app.Use(handler)
	app.Put("/test", func(c fiber.Ctx) error {
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("PUT", "/test", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("PUT with JSON should succeed, got status %d", resp.StatusCode)
	}

	reqNoJSON := httptest.NewRequest("PUT", "/test", nil)
	respNoJSON, _ := app.Test(reqNoJSON)

	if respNoJSON.StatusCode != 415 {
		t.Errorf("PUT without JSON should fail, got status %d", respNoJSON.StatusCode)
	}
}

func TestContentNegotiation_RequiresJSONForPATCH(t *testing.T) {
	handler := ContentNegotiation()

	app := fiber.New()
	app.Use(handler)
	app.Patch("/test", func(c fiber.Ctx) error {
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("PATCH", "/test", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("PATCH with JSON should succeed, got status %d", resp.StatusCode)
	}

	reqNoJSON := httptest.NewRequest("PATCH", "/test", nil)
	respNoJSON, _ := app.Test(reqNoJSON)

	if respNoJSON.StatusCode != 415 {
		t.Errorf("PATCH without JSON should fail, got status %d", respNoJSON.StatusCode)
	}
}

func TestContentNegotiation_AllowsDELETEWithoutContentType(t *testing.T) {
	handler := ContentNegotiation()

	app := fiber.New()
	app.Use(handler)
	app.Delete("/test", func(c fiber.Ctx) error {
		return c.SendStatus(204)
	})

	req := httptest.NewRequest("DELETE", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 204 {
		t.Errorf("DELETE should not require Content-Type, got status %d", resp.StatusCode)
	}
}

func TestContentNegotiation_AllMethodsIntegration(t *testing.T) {
	handler := ContentNegotiation()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error { return c.SendStatus(200) })
	app.Post("/test", func(c fiber.Ctx) error { return c.SendStatus(201) })
	app.Put("/test", func(c fiber.Ctx) error { return c.SendStatus(200) })
	app.Delete("/test", func(c fiber.Ctx) error { return c.SendStatus(204) })
	app.Patch("/test", func(c fiber.Ctx) error { return c.SendStatus(200) })

	tests := []struct {
		method         string
		contentType    string
		expectedStatus int
	}{
		{"GET", "", 200},
		{"GET", "application/json", 200},
		{"POST", "", 415},
		{"POST", "application/json", 201},
		{"PUT", "", 415},
		{"PUT", "application/json", 200},
		{"DELETE", "", 204},
		{"DELETE", "application/json", 204},
		{"PATCH", "", 415},
		{"PATCH", "application/json", 200},
	}

	for _, tt := range tests {
		t.Run(tt.method+"_"+tt.contentType, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", strings.NewReader("{}"))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			resp, _ := app.Test(req)

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("%s with content-type '%s': expected status %d, got %d",
					tt.method, tt.contentType, tt.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestContentNegotiation_ErrorMessage(t *testing.T) {
	handler := ContentNegotiation()

	app := fiber.New()
	app.Use(handler)
	app.Post("/test", func(c fiber.Ctx) error {
		return c.SendStatus(201)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != 415 {
		t.Fatalf("expected status 415, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var errorResp map[string]interface{}
	if err := json.Unmarshal(body, &errorResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}

	errorMsg, ok := errorResp["error"].(string)
	if !ok {
		t.Fatal("expected error field as string")
	}

	if !strings.Contains(errorMsg, "Content-Type") && !strings.Contains(errorMsg, "application/json") {
		t.Errorf("error message should mention Content-Type and application/json, got: %s", errorMsg)
	}
}

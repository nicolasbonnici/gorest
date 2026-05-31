package middleware

import (
	"compress/gzip"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestCompress_WithGzipEncoding(t *testing.T) {
	handler := Compress(2) // Default level

	testContent := strings.Repeat("this is a test response that should be compressed. ", 50)

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error {
		// Use a larger response to ensure compression is applied
		return c.SendString(testContent)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentEncoding := resp.Header.Get("Content-Encoding")
	if contentEncoding != "gzip" {
		t.Errorf("expected Content-Encoding 'gzip', got '%s'", contentEncoding)
	}

	// Verify the body is actually compressed
	reader, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to decompress response: %v", err)
	}

	if string(body) != testContent {
		t.Errorf("decompressed body doesn't match original content")
	}
}

func TestCompress_WithoutAcceptEncoding(t *testing.T) {
	handler := Compress(2)

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("uncompressed response")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	// No Accept-Encoding header

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentEncoding := resp.Header.Get("Content-Encoding")
	if contentEncoding != "" {
		t.Errorf("expected no Content-Encoding header, got '%s'", contentEncoding)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	expected := "uncompressed response"
	if string(body) != expected {
		t.Errorf("expected body '%s', got '%s'", expected, string(body))
	}
}

func TestCompress_DifferentLevels(t *testing.T) {
	testContent := strings.Repeat("a", 1000) // Repeatable content compresses well

	tests := []struct {
		name  string
		level int
	}{
		{"level 1 - best speed", 1},
		{"level 2 - default", 2},
		{"level 3 - best compression", 3},
		{"invalid level defaults to 2", 99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Compress(tt.level)

			app := fiber.New()
			app.Use(handler)
			app.Get("/test", func(c fiber.Ctx) error {
				return c.SendString(testContent)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Accept-Encoding", "gzip")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}

			if resp.StatusCode != 200 {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			contentEncoding := resp.Header.Get("Content-Encoding")
			if contentEncoding != "gzip" {
				t.Errorf("expected Content-Encoding 'gzip', got '%s'", contentEncoding)
			}

			// Verify we can decompress
			reader, err := gzip.NewReader(resp.Body)
			if err != nil {
				t.Fatalf("failed to create gzip reader: %v", err)
			}
			defer reader.Close()

			body, err := io.ReadAll(reader)
			if err != nil {
				t.Fatalf("failed to decompress response: %v", err)
			}

			if string(body) != testContent {
				t.Errorf("decompressed content doesn't match original")
			}
		})
	}
}

func TestCompress_MultipleEncodings(t *testing.T) {
	handler := Compress(2)

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error {
		// Use a larger response to ensure compression is applied
		return c.SendString(strings.Repeat("test response ", 100))
	})

	// Test with multiple acceptable encodings
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "deflate, gzip, br")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	contentEncoding := resp.Header.Get("Content-Encoding")
	// Should use one of the supported encodings
	if contentEncoding == "" {
		t.Errorf("expected Content-Encoding header to be set")
	}
}

func TestCompress_JSONResponse(t *testing.T) {
	handler := Compress(2)

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c fiber.Ctx) error {
		// Use a larger JSON response to ensure compression is applied
		data := make([]int, 200)
		for i := range data {
			data[i] = i
		}
		return c.JSON(fiber.Map{
			"message": "hello world " + strings.Repeat("with more data ", 20),
			"data":    data,
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	contentEncoding := resp.Header.Get("Content-Encoding")
	if contentEncoding != "gzip" {
		t.Errorf("expected Content-Encoding 'gzip', got '%s'", contentEncoding)
	}

	// Verify JSON can be decompressed
	reader, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer reader.Close()

	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to decompress response: %v", err)
	}

	// Just verify it's valid JSON-like content
	if !strings.Contains(string(body), "hello world") {
		t.Errorf("expected JSON content to contain 'hello world'")
	}
}

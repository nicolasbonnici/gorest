package ratelimit

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestNewPlugin(t *testing.T) {
	plugin := NewPlugin()

	if plugin == nil {
		t.Fatal("expected non-nil plugin")
	}

	rlPlugin, ok := plugin.(*RateLimitPlugin)
	if !ok {
		t.Fatal("expected *RateLimitPlugin type")
	}

	if rlPlugin.requestsPerSecond != 100 {
		t.Errorf("expected default requestsPerSecond 100, got %d", rlPlugin.requestsPerSecond)
	}

	if rlPlugin.burst != 200 {
		t.Errorf("expected default burst 200, got %d", rlPlugin.burst)
	}
}

func TestRateLimitPlugin_Name(t *testing.T) {
	plugin := NewPlugin()

	name := plugin.Name()

	if name != "ratelimit" {
		t.Errorf("expected plugin name 'ratelimit', got '%s'", name)
	}
}

func TestRateLimitPlugin_Initialize_EmptyConfig(t *testing.T) {
	plugin := &RateLimitPlugin{
		requestsPerSecond: 50,
		burst:             100,
	}

	err := plugin.Initialize(map[string]interface{}{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.requestsPerSecond != 50 {
		t.Errorf("expected requestsPerSecond to remain 50, got %d", plugin.requestsPerSecond)
	}

	if plugin.burst != 100 {
		t.Errorf("expected burst to remain 100, got %d", plugin.burst)
	}
}

func TestRateLimitPlugin_Initialize_WithIntValues(t *testing.T) {
	plugin := &RateLimitPlugin{}

	config := map[string]interface{}{
		"requests_per_second": 50,
		"burst":               150,
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.requestsPerSecond != 50 {
		t.Errorf("expected requestsPerSecond 50, got %d", plugin.requestsPerSecond)
	}

	if plugin.burst != 150 {
		t.Errorf("expected burst 150, got %d", plugin.burst)
	}
}

func TestRateLimitPlugin_Initialize_WithFloat64Values(t *testing.T) {
	plugin := &RateLimitPlugin{}

	config := map[string]interface{}{
		"requests_per_second": float64(30),
		"burst":               float64(60),
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.requestsPerSecond != 30 {
		t.Errorf("expected requestsPerSecond 30, got %d", plugin.requestsPerSecond)
	}

	if plugin.burst != 60 {
		t.Errorf("expected burst 60, got %d", plugin.burst)
	}
}

func TestRateLimitPlugin_Initialize_WithMixedTypes(t *testing.T) {
	plugin := &RateLimitPlugin{}

	config := map[string]interface{}{
		"requests_per_second": float64(25),
		"burst":               100,
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.requestsPerSecond != 25 {
		t.Errorf("expected requestsPerSecond 25, got %d", plugin.requestsPerSecond)
	}

	if plugin.burst != 100 {
		t.Errorf("expected burst 100, got %d", plugin.burst)
	}
}

func TestRateLimitPlugin_Initialize_OnlyRequestsPerSecond(t *testing.T) {
	plugin := &RateLimitPlugin{burst: 200}

	config := map[string]interface{}{
		"requests_per_second": 75,
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.requestsPerSecond != 75 {
		t.Errorf("expected requestsPerSecond 75, got %d", plugin.requestsPerSecond)
	}

	if plugin.burst != 200 {
		t.Errorf("expected burst to remain 200, got %d", plugin.burst)
	}
}

func TestRateLimitPlugin_Initialize_OnlyBurst(t *testing.T) {
	plugin := &RateLimitPlugin{requestsPerSecond: 100}

	config := map[string]interface{}{
		"burst": float64(300),
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.requestsPerSecond != 100 {
		t.Errorf("expected requestsPerSecond to remain 100, got %d", plugin.requestsPerSecond)
	}

	if plugin.burst != 300 {
		t.Errorf("expected burst 300, got %d", plugin.burst)
	}
}

func TestRateLimitPlugin_Initialize_InvalidTypes(t *testing.T) {
	plugin := &RateLimitPlugin{
		requestsPerSecond: 100,
		burst:             200,
	}

	config := map[string]interface{}{
		"requests_per_second": "invalid",
		"burst":               []int{1, 2, 3},
	}

	err := plugin.Initialize(config)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if plugin.requestsPerSecond != 100 {
		t.Errorf("expected requestsPerSecond to remain 100, got %d", plugin.requestsPerSecond)
	}

	if plugin.burst != 200 {
		t.Errorf("expected burst to remain 200, got %d", plugin.burst)
	}
}

func TestRateLimitPlugin_Handler_BelowLimit(t *testing.T) {
	plugin := &RateLimitPlugin{
		requestsPerSecond: 10,
		burst:             20,
	}

	handler := plugin.Handler()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.1")

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

func TestRateLimitPlugin_Handler_ExceedLimit(t *testing.T) {
	plugin := &RateLimitPlugin{
		requestsPerSecond: 3,
		burst:             5,
	}

	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	var exceededCount int
	totalRequests := 10

	for i := 0; i < totalRequests; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.100")

		resp, _ := app.Test(req)

		if resp.StatusCode == 429 {
			exceededCount++

			body, _ := io.ReadAll(resp.Body)
			var result map[string]interface{}
			json.Unmarshal(body, &result)

			if result["error"] == nil {
				t.Errorf("request %d: expected error message in 429 response", i+1)
			}

			errorMsg, ok := result["error"].(string)
			if !ok || errorMsg == "" {
				t.Errorf("request %d: expected non-empty error message", i+1)
			}
		}
	}

	if exceededCount == 0 {
		t.Error("expected at least one request to exceed rate limit")
	}
}

func TestRateLimitPlugin_Handler_TooManyRequestsResponse(t *testing.T) {
	plugin := &RateLimitPlugin{
		requestsPerSecond: 2,
		burst:             2,
	}

	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "10.0.0.1")
		app.Test(req)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	resp, _ := app.Test(req)

	if resp.StatusCode != 429 {
		t.Errorf("expected status 429, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	err := json.Unmarshal(body, &result)

	if err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	errorMsg, ok := result["error"].(string)
	if !ok {
		t.Fatal("expected error field in response")
	}

	if errorMsg != "Rate limit exceeded: 2 requests per second" {
		t.Errorf("expected 'Rate limit exceeded: 2 requests per second', got '%s'", errorMsg)
	}
}

func TestRateLimitPlugin_Handler_DifferentIPs(t *testing.T) {
	plugin := &RateLimitPlugin{
		requestsPerSecond: 5,
		burst:             10,
	}

	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", ip1)
		app.Test(req)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", ip2)
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200 for different IP, got %d", resp.StatusCode)
	}
}

func TestRateLimitPlugin_Handler_IPExtraction(t *testing.T) {
	plugin := &RateLimitPlugin{
		requestsPerSecond: 5,
		burst:             10,
	}

	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ip": c.IP()})
	})

	tests := []struct {
		name       string
		remoteAddr string
		forwarded  string
	}{
		{
			name:       "direct connection",
			remoteAddr: "192.168.1.100",
			forwarded:  "",
		},
		{
			name:       "forwarded header",
			remoteAddr: "10.0.0.1",
			forwarded:  "203.0.113.1",
		},
		{
			name:       "multiple forwarded",
			remoteAddr: "10.0.0.1",
			forwarded:  "203.0.113.1, 198.51.100.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.forwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.forwarded)
			}

			resp, _ := app.Test(req)

			if resp.StatusCode != 200 {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
			}
		})
	}
}

func TestRateLimitPlugin_Handler_RateLimitReset(t *testing.T) {
	plugin := &RateLimitPlugin{
		requestsPerSecond: 5,
		burst:             5,
	}

	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "172.16.0.1")
		app.Test(req)
	}

	time.Sleep(1100 * time.Millisecond)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "172.16.0.1")
	resp, _ := app.Test(req)

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200 after reset, got %d", resp.StatusCode)
	}
}

func TestRateLimitPlugin_Handler_ConcurrentRequests(t *testing.T) {
	plugin := &RateLimitPlugin{
		requestsPerSecond: 10,
		burst:             20,
	}

	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	successCount := 0
	for i := 0; i < 15; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "10.10.10.10")

		resp, _ := app.Test(req)

		if resp.StatusCode == 200 {
			successCount++
		}
	}

	if successCount == 0 {
		t.Error("expected at least some requests to succeed")
	}
}

func TestRateLimitPlugin_Handler_LowLimit(t *testing.T) {
	plugin := &RateLimitPlugin{
		requestsPerSecond: 1,
		burst:             1,
	}

	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.Header.Set("X-Forwarded-For", "192.0.2.1")
	resp1, _ := app.Test(req1)

	if resp1.StatusCode != 200 {
		t.Errorf("first request: expected status 200, got %d", resp1.StatusCode)
	}

	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.Header.Set("X-Forwarded-For", "192.0.2.1")
	resp2, _ := app.Test(req2)

	if resp2.StatusCode != 429 {
		t.Errorf("second request: expected status 429, got %d", resp2.StatusCode)
	}
}

func TestRateLimitPlugin_Handler_HighLimit(t *testing.T) {
	plugin := &RateLimitPlugin{
		requestsPerSecond: 1000,
		burst:             2000,
	}

	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "198.18.0.1")

		resp, _ := app.Test(req)

		if resp.StatusCode != 200 {
			t.Errorf("request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}
	}
}

func TestRateLimitPlugin_IntegrationWithFiber(t *testing.T) {
	plugin := NewPlugin()

	config := map[string]interface{}{
		"requests_per_second": float64(10),
		"burst":               20,
	}

	err := plugin.Initialize(config)
	if err != nil {
		t.Fatalf("failed to initialize plugin: %v", err)
	}

	app := fiber.New()
	app.Use(plugin.Handler())
	app.Get("/api/data", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "success"})
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/api/data", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.100")

		resp, _ := app.Test(req)

		if resp.StatusCode != 200 {
			t.Errorf("request %d: expected status 200, got %d", i+1, resp.StatusCode)
		}
	}
}

func TestRateLimitPlugin_Handler_MultipleEndpoints(t *testing.T) {
	plugin := &RateLimitPlugin{
		requestsPerSecond: 3,
		burst:             3,
	}

	handler := plugin.Handler()

	app := fiber.New()
	app.Use(handler)
	app.Get("/endpoint1", func(c *fiber.Ctx) error { return c.SendString("endpoint1") })
	app.Get("/endpoint2", func(c *fiber.Ctx) error { return c.SendString("endpoint2") })

	for i := 0; i < 4; i++ {
		req := httptest.NewRequest("GET", "/endpoint1", nil)
		req.Header.Set("X-Forwarded-For", "198.51.100.1")
		app.Test(req)
	}

	req := httptest.NewRequest("GET", "/endpoint2", nil)
	req.Header.Set("X-Forwarded-For", "198.51.100.1")
	resp, _ := app.Test(req)

	if resp.StatusCode != 429 {
		t.Errorf("expected rate limit to apply across endpoints, got status %d", resp.StatusCode)
	}
}

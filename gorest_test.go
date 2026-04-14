package gorest

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestVersionedRouting_WithVersion(t *testing.T) {
	// Save original version and restore after test
	originalVersion := Version
	defer func() { Version = originalVersion }()

	// Set version to simulate production build
	Version = "v2.5.3"

	app := fiber.New()

	// Create version group as done in Start()
	apiVersion := Version
	if apiVersion == "" || apiVersion == "dev" {
		apiVersion = "v1.0.0"
	}
	versionedRouter := app.Group("/" + apiVersion)

	// Register a test route on the versioned router
	versionedRouter.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Test versioned route
	req := httptest.NewRequest("GET", "/v2.5.3/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test versioned route: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for versioned route, got %d", resp.StatusCode)
	}

	// Test unversioned route should return 404
	req = httptest.NewRequest("GET", "/test", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test unversioned route: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 for unversioned route when version is set, got %d", resp.StatusCode)
	}
}

func TestVersionedRouting_DevModeFallback(t *testing.T) {
	// Save original version and restore after test
	originalVersion := Version
	defer func() { Version = originalVersion }()

	// Set version to dev (should fallback to v1.0.0)
	Version = "dev"

	app := fiber.New()

	// Create version group as done in Start()
	apiVersion := Version
	if apiVersion == "" || apiVersion == "dev" {
		apiVersion = "v1.0.0"
	}
	versionedRouter := app.Group("/" + apiVersion)

	// Register a test route on the versioned router
	versionedRouter.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Test fallback version route (should use v1.0.0)
	req := httptest.NewRequest("GET", "/v1.0.0/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test fallback versioned route: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for /v1.0.0/test route in dev mode, got %d", resp.StatusCode)
	}

	// Test /dev/test should return 404 (fallback is v1.0.0, not dev)
	req = httptest.NewRequest("GET", "/dev/test", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test /dev/test route: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 for /dev/test route, got %d", resp.StatusCode)
	}

	// Test unversioned route should return 404
	req = httptest.NewRequest("GET", "/test", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test unversioned route: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 for unversioned route, got %d", resp.StatusCode)
	}
}

func TestVersionedRouting_EmptyVersionFallback(t *testing.T) {
	// Save original version and restore after test
	originalVersion := Version
	defer func() { Version = originalVersion }()

	// Set version to empty (should fallback to v1.0.0)
	Version = ""

	app := fiber.New()

	// Create version group as done in Start()
	apiVersion := Version
	if apiVersion == "" || apiVersion == "dev" {
		apiVersion = "v1.0.0"
	}
	versionedRouter := app.Group("/" + apiVersion)

	// Register a test route on the versioned router
	versionedRouter.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	// Test fallback version route (should use v1.0.0)
	req := httptest.NewRequest("GET", "/v1.0.0/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test fallback route: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for /v1.0.0/test with empty version, got %d", resp.StatusCode)
	}

	// Test unversioned route should return 404
	req = httptest.NewRequest("GET", "/test", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test unversioned route: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 for unversioned route, got %d", resp.StatusCode)
	}
}

func TestVersionedRouting_PluginEndpoints(t *testing.T) {
	// Save original version and restore after test
	originalVersion := Version
	defer func() { Version = originalVersion }()

	// Set version to simulate production build
	Version = "v2.5.3"

	app := fiber.New()

	// Create version group as done in Start()
	apiVersion := Version
	if apiVersion == "" || apiVersion == "dev" {
		apiVersion = "v1.0.0"
	}
	versionedRouter := app.Group("/" + apiVersion)

	// Simulate plugin endpoint setup
	versionedRouter.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("healthy")
	})

	versionedRouter.Post("/login", func(c *fiber.Ctx) error {
		return c.SendString("logged in")
	})

	// Test versioned health endpoint
	req := httptest.NewRequest("GET", "/v2.5.3/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test versioned health endpoint: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for versioned health endpoint, got %d", resp.StatusCode)
	}

	// Test versioned login endpoint
	req = httptest.NewRequest("POST", "/v2.5.3/login", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test versioned login endpoint: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for versioned login endpoint, got %d", resp.StatusCode)
	}

	// Test unversioned endpoints should return 404
	req = httptest.NewRequest("GET", "/health", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test unversioned health endpoint: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 for unversioned health endpoint, got %d", resp.StatusCode)
	}

	req = httptest.NewRequest("POST", "/login", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test unversioned login endpoint: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 for unversioned login endpoint, got %d", resp.StatusCode)
	}
}

func TestVersionedRouting_PluginEndpointsWithFallback(t *testing.T) {
	// Save original version and restore after test
	originalVersion := Version
	defer func() { Version = originalVersion }()

	// Set version to dev (should fallback to v1.0.0)
	Version = "dev"

	app := fiber.New()

	// Create version group as done in Start()
	apiVersion := Version
	if apiVersion == "" || apiVersion == "dev" {
		apiVersion = "v1.0.0"
	}
	versionedRouter := app.Group("/" + apiVersion)

	// Simulate plugin endpoint setup
	versionedRouter.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("healthy")
	})

	versionedRouter.Post("/login", func(c *fiber.Ctx) error {
		return c.SendString("logged in")
	})

	// Test versioned health endpoint with v1.0.0 fallback
	req := httptest.NewRequest("GET", "/v1.0.0/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test versioned health endpoint: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for /v1.0.0/health endpoint, got %d", resp.StatusCode)
	}

	// Test versioned login endpoint with v1.0.0 fallback
	req = httptest.NewRequest("POST", "/v1.0.0/login", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test versioned login endpoint: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for /v1.0.0/login endpoint, got %d", resp.StatusCode)
	}

	// Test unversioned endpoints should return 404
	req = httptest.NewRequest("GET", "/health", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test unversioned health endpoint: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 for unversioned health endpoint, got %d", resp.StatusCode)
	}

	// Test /dev prefix should return 404 (we use v1.0.0, not dev)
	req = httptest.NewRequest("GET", "/dev/health", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test /dev/health: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 for /dev/health, got %d", resp.StatusCode)
	}
}

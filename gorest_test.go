package gorest

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestVersionedRouting_WithVersion(t *testing.T) {
	originalVersion := Version
	defer func() { Version = originalVersion }()

	Version = "v2.5.3"

	app := fiber.New()

	apiVersion := Version
	if apiVersion == "" || apiVersion == "dev" {
		apiVersion = "v1.0.0"
	}
	versionedRouter := app.Group("/" + apiVersion)

	versionedRouter.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/v2.5.3/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test versioned route: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for versioned route, got %d", resp.StatusCode)
	}

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
	originalVersion := Version
	defer func() { Version = originalVersion }()

	Version = "dev"

	app := fiber.New()

	apiVersion := Version
	if apiVersion == "" || apiVersion == "dev" {
		apiVersion = "v1.0.0"
	}
	versionedRouter := app.Group("/" + apiVersion)

	versionedRouter.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/v1.0.0/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test fallback versioned route: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for /v1.0.0/test route in dev mode, got %d", resp.StatusCode)
	}

	req = httptest.NewRequest("GET", "/dev/test", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test /dev/test route: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 for /dev/test route, got %d", resp.StatusCode)
	}

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
	originalVersion := Version
	defer func() { Version = originalVersion }()

	Version = ""

	app := fiber.New()

	apiVersion := Version
	if apiVersion == "" || apiVersion == "dev" {
		apiVersion = "v1.0.0"
	}
	versionedRouter := app.Group("/" + apiVersion)

	versionedRouter.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/v1.0.0/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test fallback route: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for /v1.0.0/test with empty version, got %d", resp.StatusCode)
	}

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
	originalVersion := Version
	defer func() { Version = originalVersion }()

	Version = "v2.5.3"

	app := fiber.New()

	apiVersion := Version
	if apiVersion == "" || apiVersion == "dev" {
		apiVersion = "v1.0.0"
	}
	versionedRouter := app.Group("/" + apiVersion)

	versionedRouter.Get("/health", func(c fiber.Ctx) error {
		return c.SendString("healthy")
	})

	versionedRouter.Post("/login", func(c fiber.Ctx) error {
		return c.SendString("logged in")
	})

	req := httptest.NewRequest("GET", "/v2.5.3/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test versioned health endpoint: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for versioned health endpoint, got %d", resp.StatusCode)
	}

	req = httptest.NewRequest("POST", "/v2.5.3/login", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test versioned login endpoint: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for versioned login endpoint, got %d", resp.StatusCode)
	}

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
	originalVersion := Version
	defer func() { Version = originalVersion }()

	Version = "dev"

	app := fiber.New()

	apiVersion := Version
	if apiVersion == "" || apiVersion == "dev" {
		apiVersion = "v1.0.0"
	}
	versionedRouter := app.Group("/" + apiVersion)

	versionedRouter.Get("/health", func(c fiber.Ctx) error {
		return c.SendString("healthy")
	})

	versionedRouter.Post("/login", func(c fiber.Ctx) error {
		return c.SendString("logged in")
	})

	req := httptest.NewRequest("GET", "/v1.0.0/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test versioned health endpoint: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for /v1.0.0/health endpoint, got %d", resp.StatusCode)
	}

	req = httptest.NewRequest("POST", "/v1.0.0/login", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test versioned login endpoint: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for /v1.0.0/login endpoint, got %d", resp.StatusCode)
	}

	req = httptest.NewRequest("GET", "/health", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test unversioned health endpoint: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 for unversioned health endpoint, got %d", resp.StatusCode)
	}

	req = httptest.NewRequest("GET", "/dev/health", nil)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test /dev/health: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 for /dev/health, got %d", resp.StatusCode)
	}
}

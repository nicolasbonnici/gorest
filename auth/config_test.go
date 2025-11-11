package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultAuthConfig(t *testing.T) {
	cfg := DefaultAuthConfig()

	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}

	if cfg.RequireAuth == nil {
		t.Fatal("Expected RequireAuth map to be initialized")
	}

	// Check that users resource requires all methods
	usersMethods := cfg.RequireAuth["users"]
	expectedMethods := []string{"GET", "POST", "PUT", "DELETE"}

	if len(usersMethods) != len(expectedMethods) {
		t.Errorf("Expected %d methods for users, got %d", len(expectedMethods), len(usersMethods))
	}

	for _, method := range expectedMethods {
		found := false
		for _, m := range usersMethods {
			if m == method {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected method %s for users resource", method)
		}
	}

	// Check that todos resource requires all methods
	todosMethods := cfg.RequireAuth["todos"]
	if len(todosMethods) != len(expectedMethods) {
		t.Errorf("Expected %d methods for todos, got %d", len(expectedMethods), len(todosMethods))
	}
}

func TestNoAuthConfig(t *testing.T) {
	cfg := NoAuthConfig()

	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}

	if cfg.RequireAuth == nil {
		t.Fatal("Expected RequireAuth map to be initialized")
	}

	if len(cfg.RequireAuth) != 0 {
		t.Errorf("Expected empty RequireAuth map, got %d entries", len(cfg.RequireAuth))
	}
}

func TestLoadAuthConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "auth.json")

	configContent := `{
  "require_auth": {
    "users": ["GET", "POST"],
    "todos": ["DELETE"]
  }
}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load the config
	cfg, err := LoadAuthConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}

	// Verify users resource
	usersMethods := cfg.RequireAuth["users"]
	if len(usersMethods) != 2 {
		t.Errorf("Expected 2 methods for users, got %d", len(usersMethods))
	}

	// Verify todos resource
	todosMethods := cfg.RequireAuth["todos"]
	if len(todosMethods) != 1 {
		t.Errorf("Expected 1 method for todos, got %d", len(todosMethods))
	}
	if todosMethods[0] != "DELETE" {
		t.Errorf("Expected DELETE method for todos, got %s", todosMethods[0])
	}

	// Test loading non-existent file
	_, err = LoadAuthConfig(filepath.Join(tmpDir, "nonexistent.json"))
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}

	// Test loading invalid JSON
	invalidPath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	_, err = LoadAuthConfig(invalidPath)
	if err == nil {
		t.Error("Expected error when loading invalid JSON")
	}
}

func TestSaveAuthConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "auth.json")

	cfg := &AuthConfig{
		RequireAuth: map[string][]string{
			"users": {"GET", "POST"},
			"todos": {"DELETE"},
		},
	}

	// Save the config
	if err := SaveAuthConfig(cfg, configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Load it back and verify
	loadedCfg, err := LoadAuthConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if len(loadedCfg.RequireAuth) != len(cfg.RequireAuth) {
		t.Errorf("Expected %d resources, got %d", len(cfg.RequireAuth), len(loadedCfg.RequireAuth))
	}

	// Verify content matches
	for resource, methods := range cfg.RequireAuth {
		loadedMethods, ok := loadedCfg.RequireAuth[resource]
		if !ok {
			t.Errorf("Resource %s not found in loaded config", resource)
			continue
		}

		if len(loadedMethods) != len(methods) {
			t.Errorf("Expected %d methods for %s, got %d", len(methods), resource, len(loadedMethods))
		}
	}
}

func TestRequiresAuth(t *testing.T) {
	cfg := &AuthConfig{
		RequireAuth: map[string][]string{
			"users": {"GET", "POST", "PUT"},
			"todos": {"DELETE"},
		},
	}

	tests := []struct {
		name     string
		resource string
		method   string
		expected bool
	}{
		{
			name:     "users GET requires auth",
			resource: "users",
			method:   "GET",
			expected: true,
		},
		{
			name:     "users POST requires auth",
			resource: "users",
			method:   "POST",
			expected: true,
		},
		{
			name:     "users DELETE does not require auth",
			resource: "users",
			method:   "DELETE",
			expected: false,
		},
		{
			name:     "todos DELETE requires auth",
			resource: "todos",
			method:   "DELETE",
			expected: true,
		},
		{
			name:     "todos GET does not require auth",
			resource: "todos",
			method:   "GET",
			expected: false,
		},
		{
			name:     "non-existent resource",
			resource: "nonexistent",
			method:   "GET",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cfg.RequiresAuth(tt.resource, tt.method)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestSetResourceAuth(t *testing.T) {
	cfg := &AuthConfig{
		RequireAuth: make(map[string][]string),
	}

	// Test setting auth for a new resource
	cfg.SetResourceAuth("users", []string{"GET", "POST"})

	usersMethods := cfg.RequireAuth["users"]
	if len(usersMethods) != 2 {
		t.Errorf("Expected 2 methods, got %d", len(usersMethods))
	}

	// Test updating auth for existing resource
	cfg.SetResourceAuth("users", []string{"GET"})

	usersMethods = cfg.RequireAuth["users"]
	if len(usersMethods) != 1 {
		t.Errorf("Expected 1 method after update, got %d", len(usersMethods))
	}

	// Test setting auth when RequireAuth is nil
	cfgNil := &AuthConfig{}
	cfgNil.SetResourceAuth("todos", []string{"DELETE"})

	if cfgNil.RequireAuth == nil {
		t.Error("Expected RequireAuth map to be initialized")
	}

	if len(cfgNil.RequireAuth["todos"]) != 1 {
		t.Errorf("Expected 1 method for todos, got %d", len(cfgNil.RequireAuth["todos"]))
	}
}

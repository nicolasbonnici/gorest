package plugin

import (
	"testing"

	"github.com/gofiber/fiber/v3"
)

type mockPlugin struct {
	name    string
	handler fiber.Handler
}

func (m *mockPlugin) Name() string {
	return m.name
}

func (m *mockPlugin) Initialize(config map[string]interface{}) error {
	return nil
}

func (m *mockPlugin) Handler() fiber.Handler {
	return m.handler
}

func TestNewPluginRegistry(t *testing.T) {
	registry := NewPluginRegistry()

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	if registry.plugins == nil {
		t.Error("expected non-nil plugins map")
	}

	if len(registry.plugins) != 0 {
		t.Errorf("expected empty plugins map, got %d items", len(registry.plugins))
	}
}

func TestRegister_Single(t *testing.T) {
	registry := NewPluginRegistry()
	plugin := &mockPlugin{
		name: "test-plugin",
		handler: func(c fiber.Ctx) error {
			return c.Next()
		},
	}

	registry.Register(plugin)

	if len(registry.plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(registry.plugins))
	}

	stored, exists := registry.plugins["test-plugin"]
	if !exists {
		t.Fatal("expected plugin to be stored with name 'test-plugin'")
	}

	if stored.Name() != "test-plugin" {
		t.Errorf("expected plugin name 'test-plugin', got '%s'", stored.Name())
	}
}

func TestRegister_Multiple(t *testing.T) {
	registry := NewPluginRegistry()
	plugin1 := &mockPlugin{
		name: "plugin1",
		handler: func(c fiber.Ctx) error {
			return c.Next()
		},
	}
	plugin2 := &mockPlugin{
		name: "plugin2",
		handler: func(c fiber.Ctx) error {
			return c.Next()
		},
	}
	plugin3 := &mockPlugin{
		name: "plugin3",
		handler: func(c fiber.Ctx) error {
			return c.Next()
		},
	}

	registry.Register(plugin1)
	registry.Register(plugin2)
	registry.Register(plugin3)

	if len(registry.plugins) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(registry.plugins))
	}

	for _, name := range []string{"plugin1", "plugin2", "plugin3"} {
		stored, exists := registry.plugins[name]
		if !exists {
			t.Errorf("expected plugin '%s' to be stored", name)
		}
		if stored.Name() != name {
			t.Errorf("expected plugin name '%s', got '%s'", name, stored.Name())
		}
	}
}

func TestRegister_Overwrite(t *testing.T) {
	registry := NewPluginRegistry()
	plugin1 := &mockPlugin{name: "test", handler: func(c fiber.Ctx) error { return c.Next() }}
	plugin2 := &mockPlugin{name: "test", handler: func(c fiber.Ctx) error { return c.SendStatus(200) }}

	registry.Register(plugin1)
	registry.Register(plugin2)

	if len(registry.plugins) != 1 {
		t.Fatalf("expected 1 plugin (overwritten), got %d", len(registry.plugins))
	}

	stored, exists := registry.plugins["test"]
	if !exists {
		t.Fatal("expected plugin 'test' to exist")
	}

	if stored != plugin2 {
		t.Error("expected second plugin to overwrite first plugin")
	}
}

func TestGet_Exists(t *testing.T) {
	registry := NewPluginRegistry()
	plugin := &mockPlugin{name: "test-plugin", handler: func(c fiber.Ctx) error { return c.Next() }}

	registry.Register(plugin)

	retrieved, exists := registry.Get("test-plugin")

	if !exists {
		t.Fatal("expected plugin to exist")
	}

	if retrieved == nil {
		t.Fatal("expected non-nil plugin")
	}

	if retrieved.Name() != "test-plugin" {
		t.Errorf("expected plugin name 'test-plugin', got '%s'", retrieved.Name())
	}
}

func TestGet_NotExists(t *testing.T) {
	registry := NewPluginRegistry()

	retrieved, exists := registry.Get("nonexistent")

	if exists {
		t.Error("expected plugin not to exist")
	}

	if retrieved != nil {
		t.Error("expected nil plugin")
	}
}

func TestGet_MultiplePlugins(t *testing.T) {
	registry := NewPluginRegistry()
	registry.Register(&mockPlugin{name: "plugin1", handler: func(c fiber.Ctx) error { return c.Next() }})
	registry.Register(&mockPlugin{name: "plugin2", handler: func(c fiber.Ctx) error { return c.Next() }})
	registry.Register(&mockPlugin{name: "plugin3", handler: func(c fiber.Ctx) error { return c.Next() }})

	tests := []struct {
		name   string
		exists bool
	}{
		{"plugin1", true},
		{"plugin2", true},
		{"plugin3", true},
		{"plugin4", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin, exists := registry.Get(tt.name)
			if exists != tt.exists {
				t.Errorf("expected exists=%v, got %v", tt.exists, exists)
			}
			if tt.exists && plugin.Name() != tt.name {
				t.Errorf("expected plugin name '%s', got '%s'", tt.name, plugin.Name())
			}
		})
	}
}

func TestGetAll_Empty(t *testing.T) {
	registry := NewPluginRegistry()
	plugins := registry.GetAll()

	if plugins == nil {
		t.Fatal("expected non-nil map")
	}

	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestGetAll_WithPlugins(t *testing.T) {
	registry := NewPluginRegistry()
	plugin1 := &mockPlugin{name: "plugin1", handler: func(c fiber.Ctx) error { return c.Next() }}
	plugin2 := &mockPlugin{name: "plugin2", handler: func(c fiber.Ctx) error { return c.Next() }}

	registry.Register(plugin1)
	registry.Register(plugin2)

	plugins := registry.GetAll()

	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}

	if _, exists := plugins["plugin1"]; !exists {
		t.Error("expected 'plugin1' to exist")
	}
	if _, exists := plugins["plugin2"]; !exists {
		t.Error("expected 'plugin2' to exist")
	}
}

func TestPluginRegistry_IntegrationScenario(t *testing.T) {
	registry := NewPluginRegistry()

	logger := &mockPlugin{
		name:    "logger",
		handler: func(c fiber.Ctx) error { return c.Next() },
	}
	cors := &mockPlugin{
		name:    "cors",
		handler: func(c fiber.Ctx) error { return c.Next() },
	}
	auth := &mockPlugin{
		name:    "auth",
		handler: func(c fiber.Ctx) error { return c.Next() },
	}
	ratelimit := &mockPlugin{
		name:    "ratelimit",
		handler: func(c fiber.Ctx) error { return c.Next() },
	}

	registry.Register(logger)
	registry.Register(cors)
	registry.Register(auth)
	registry.Register(ratelimit)

	if len(registry.GetAll()) != 4 {
		t.Errorf("expected 4 plugins, got %d", len(registry.GetAll()))
	}

	if plugin, exists := registry.Get("auth"); !exists || plugin.Name() != "auth" {
		t.Error("expected auth plugin to exist")
	}

	if plugin, exists := registry.Get("ratelimit"); !exists || plugin.Name() != "ratelimit" {
		t.Error("expected ratelimit plugin to exist")
	}
}

func TestPluginHandler_NilHandler(t *testing.T) {
	plugin := &mockPlugin{
		name:    "nil-handler",
		handler: nil,
	}

	handler := plugin.Handler()

	if handler != nil {
		t.Error("expected nil handler")
	}
}

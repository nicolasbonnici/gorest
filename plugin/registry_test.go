package plugin

import (
	"testing"

	"github.com/gofiber/fiber/v2"
)

type mockGlobalPlugin struct {
	name    string
	handler fiber.Handler
}

func (m *mockGlobalPlugin) Name() string {
	return m.name
}

func (m *mockGlobalPlugin) Initialize(config map[string]interface{}) error {
	return nil
}

func (m *mockGlobalPlugin) Handler() fiber.Handler {
	return m.handler
}

type mockRoutePlugin struct {
	name string
}

func (m *mockRoutePlugin) Name() string {
	return m.name
}

func (m *mockRoutePlugin) Initialize(config map[string]interface{}) error {
	return nil
}

func (m *mockRoutePlugin) Wrap(handler fiber.Handler) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return handler(c)
	}
}

func TestNewPluginRegistry(t *testing.T) {
	registry := NewPluginRegistry()

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	if registry.globalPlugins == nil {
		t.Error("expected non-nil globalPlugins slice")
	}

	if registry.routePlugins == nil {
		t.Error("expected non-nil routePlugins map")
	}

	if len(registry.globalPlugins) != 0 {
		t.Errorf("expected empty globalPlugins slice, got %d items", len(registry.globalPlugins))
	}

	if len(registry.routePlugins) != 0 {
		t.Errorf("expected empty routePlugins map, got %d items", len(registry.routePlugins))
	}
}

func TestRegisterGlobal_Single(t *testing.T) {
	registry := NewPluginRegistry()
	plugin := &mockGlobalPlugin{
		name: "test-global",
		handler: func(c *fiber.Ctx) error {
			return c.Next()
		},
	}

	registry.RegisterGlobal(plugin)

	if len(registry.globalPlugins) != 1 {
		t.Fatalf("expected 1 global plugin, got %d", len(registry.globalPlugins))
	}

	if registry.globalPlugins[0].Name() != "test-global" {
		t.Errorf("expected plugin name 'test-global', got '%s'", registry.globalPlugins[0].Name())
	}
}

func TestRegisterGlobal_Multiple(t *testing.T) {
	registry := NewPluginRegistry()
	plugin1 := &mockGlobalPlugin{
		name: "plugin1",
		handler: func(c *fiber.Ctx) error {
			return c.Next()
		},
	}
	plugin2 := &mockGlobalPlugin{
		name: "plugin2",
		handler: func(c *fiber.Ctx) error {
			return c.Next()
		},
	}
	plugin3 := &mockGlobalPlugin{
		name: "plugin3",
		handler: func(c *fiber.Ctx) error {
			return c.Next()
		},
	}

	registry.RegisterGlobal(plugin1)
	registry.RegisterGlobal(plugin2)
	registry.RegisterGlobal(plugin3)

	if len(registry.globalPlugins) != 3 {
		t.Fatalf("expected 3 global plugins, got %d", len(registry.globalPlugins))
	}

	if registry.globalPlugins[0].Name() != "plugin1" {
		t.Errorf("expected first plugin 'plugin1', got '%s'", registry.globalPlugins[0].Name())
	}
	if registry.globalPlugins[1].Name() != "plugin2" {
		t.Errorf("expected second plugin 'plugin2', got '%s'", registry.globalPlugins[1].Name())
	}
	if registry.globalPlugins[2].Name() != "plugin3" {
		t.Errorf("expected third plugin 'plugin3', got '%s'", registry.globalPlugins[2].Name())
	}
}

func TestRegisterGlobal_OrderPreservation(t *testing.T) {
	registry := NewPluginRegistry()

	for i := 1; i <= 10; i++ {
		plugin := &mockGlobalPlugin{
			name: string(rune('A' + i - 1)),
			handler: func(c *fiber.Ctx) error {
				return c.Next()
			},
		}
		registry.RegisterGlobal(plugin)
	}

	if len(registry.globalPlugins) != 10 {
		t.Fatalf("expected 10 plugins, got %d", len(registry.globalPlugins))
	}

	for i := 0; i < 10; i++ {
		expected := string(rune('A' + i))
		if registry.globalPlugins[i].Name() != expected {
			t.Errorf("expected plugin at index %d to be '%s', got '%s'", i, expected, registry.globalPlugins[i].Name())
		}
	}
}

func TestRegisterRoute_Single(t *testing.T) {
	registry := NewPluginRegistry()
	plugin := &mockRoutePlugin{name: "test-route"}

	registry.RegisterRoute(plugin)

	if len(registry.routePlugins) != 1 {
		t.Fatalf("expected 1 route plugin, got %d", len(registry.routePlugins))
	}

	stored, exists := registry.routePlugins["test-route"]
	if !exists {
		t.Fatal("expected plugin to be stored with name 'test-route'")
	}

	if stored.Name() != "test-route" {
		t.Errorf("expected plugin name 'test-route', got '%s'", stored.Name())
	}
}

func TestRegisterRoute_Multiple(t *testing.T) {
	registry := NewPluginRegistry()
	plugins := []*mockRoutePlugin{
		{name: "auth"},
		{name: "cors"},
		{name: "rate-limiter"},
	}

	for _, p := range plugins {
		registry.RegisterRoute(p)
	}

	if len(registry.routePlugins) != 3 {
		t.Fatalf("expected 3 route plugins, got %d", len(registry.routePlugins))
	}

	for _, p := range plugins {
		stored, exists := registry.routePlugins[p.Name()]
		if !exists {
			t.Errorf("expected plugin '%s' to be stored", p.Name())
		}
		if stored.Name() != p.Name() {
			t.Errorf("expected plugin name '%s', got '%s'", p.Name(), stored.Name())
		}
	}
}

func TestRegisterRoute_Overwrite(t *testing.T) {
	registry := NewPluginRegistry()
	plugin1 := &mockRoutePlugin{name: "auth"}
	plugin2 := &mockRoutePlugin{name: "auth"}

	registry.RegisterRoute(plugin1)
	registry.RegisterRoute(plugin2)

	if len(registry.routePlugins) != 1 {
		t.Fatalf("expected 1 route plugin (overwritten), got %d", len(registry.routePlugins))
	}

	stored, exists := registry.routePlugins["auth"]
	if !exists {
		t.Fatal("expected plugin 'auth' to exist")
	}

	if stored != plugin2 {
		t.Error("expected second plugin to overwrite first plugin")
	}
}

func TestGetGlobalPlugins_Empty(t *testing.T) {
	registry := NewPluginRegistry()
	plugins := registry.GetGlobalPlugins()

	if plugins == nil {
		t.Fatal("expected non-nil slice")
	}

	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestGetGlobalPlugins_WithPlugins(t *testing.T) {
	registry := NewPluginRegistry()
	plugin1 := &mockGlobalPlugin{name: "plugin1", handler: func(c *fiber.Ctx) error { return c.Next() }}
	plugin2 := &mockGlobalPlugin{name: "plugin2", handler: func(c *fiber.Ctx) error { return c.Next() }}

	registry.RegisterGlobal(plugin1)
	registry.RegisterGlobal(plugin2)

	plugins := registry.GetGlobalPlugins()

	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}

	if plugins[0].Name() != "plugin1" {
		t.Errorf("expected first plugin 'plugin1', got '%s'", plugins[0].Name())
	}
	if plugins[1].Name() != "plugin2" {
		t.Errorf("expected second plugin 'plugin2', got '%s'", plugins[1].Name())
	}
}

func TestGetRoutePlugins_Empty(t *testing.T) {
	registry := NewPluginRegistry()
	plugins := registry.GetRoutePlugins()

	if plugins == nil {
		t.Fatal("expected non-nil map")
	}

	if len(plugins) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(plugins))
	}
}

func TestGetRoutePlugins_WithPlugins(t *testing.T) {
	registry := NewPluginRegistry()
	plugin1 := &mockRoutePlugin{name: "auth"}
	plugin2 := &mockRoutePlugin{name: "cors"}

	registry.RegisterRoute(plugin1)
	registry.RegisterRoute(plugin2)

	plugins := registry.GetRoutePlugins()

	if len(plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(plugins))
	}

	if _, exists := plugins["auth"]; !exists {
		t.Error("expected 'auth' plugin to exist")
	}
	if _, exists := plugins["cors"]; !exists {
		t.Error("expected 'cors' plugin to exist")
	}
}

func TestGetRoutePlugin_Exists(t *testing.T) {
	registry := NewPluginRegistry()
	plugin := &mockRoutePlugin{name: "test-plugin"}

	registry.RegisterRoute(plugin)

	retrieved, exists := registry.GetRoutePlugin("test-plugin")

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

func TestGetRoutePlugin_NotExists(t *testing.T) {
	registry := NewPluginRegistry()

	retrieved, exists := registry.GetRoutePlugin("nonexistent")

	if exists {
		t.Error("expected plugin not to exist")
	}

	if retrieved != nil {
		t.Error("expected nil plugin")
	}
}

func TestGetRoutePlugin_MultiplePlugins(t *testing.T) {
	registry := NewPluginRegistry()
	registry.RegisterRoute(&mockRoutePlugin{name: "plugin1"})
	registry.RegisterRoute(&mockRoutePlugin{name: "plugin2"})
	registry.RegisterRoute(&mockRoutePlugin{name: "plugin3"})

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
			plugin, exists := registry.GetRoutePlugin(tt.name)
			if exists != tt.exists {
				t.Errorf("expected exists=%v, got %v", tt.exists, exists)
			}
			if tt.exists && plugin.Name() != tt.name {
				t.Errorf("expected plugin name '%s', got '%s'", tt.name, plugin.Name())
			}
		})
	}
}

func TestApplyGlobal_EmptyRegistry(t *testing.T) {
	registry := NewPluginRegistry()
	app := fiber.New()

	err := registry.ApplyGlobal(app)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestApplyGlobal_SinglePlugin(t *testing.T) {
	registry := NewPluginRegistry()
	app := fiber.New()

	plugin := &mockGlobalPlugin{
		name: "test",
		handler: func(c *fiber.Ctx) error {
			return c.Next()
		},
	}

	registry.RegisterGlobal(plugin)
	err := registry.ApplyGlobal(app)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestApplyGlobal_MultiplePlugins(t *testing.T) {
	registry := NewPluginRegistry()
	app := fiber.New()

	plugin1 := &mockGlobalPlugin{
		name:    "plugin1",
		handler: func(c *fiber.Ctx) error { return c.Next() },
	}
	plugin2 := &mockGlobalPlugin{
		name:    "plugin2",
		handler: func(c *fiber.Ctx) error { return c.Next() },
	}

	registry.RegisterGlobal(plugin1)
	registry.RegisterGlobal(plugin2)

	err := registry.ApplyGlobal(app)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestApplyGlobal_NilHandler(t *testing.T) {
	registry := NewPluginRegistry()
	app := fiber.New()

	plugin := &mockGlobalPlugin{
		name:    "nil-handler",
		handler: nil,
	}

	registry.RegisterGlobal(plugin)
	err := registry.ApplyGlobal(app)

	if err == nil {
		t.Fatal("expected error for nil handler")
	}

	expectedMsg := "global plugin 'nil-handler' returned nil handler"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestApplyGlobal_FirstPluginNilHandler(t *testing.T) {
	registry := NewPluginRegistry()
	app := fiber.New()

	plugin1 := &mockGlobalPlugin{
		name:    "nil-plugin",
		handler: nil,
	}
	plugin2 := &mockGlobalPlugin{
		name:    "valid-plugin",
		handler: func(c *fiber.Ctx) error { return c.Next() },
	}

	registry.RegisterGlobal(plugin1)
	registry.RegisterGlobal(plugin2)

	err := registry.ApplyGlobal(app)

	if err == nil {
		t.Fatal("expected error for nil handler")
	}

	if err.Error() != "global plugin 'nil-plugin' returned nil handler" {
		t.Errorf("expected error about nil-plugin, got '%s'", err.Error())
	}
}

func TestApplyGlobal_MiddlePluginNilHandler(t *testing.T) {
	registry := NewPluginRegistry()
	app := fiber.New()

	plugin1 := &mockGlobalPlugin{
		name:    "valid1",
		handler: func(c *fiber.Ctx) error { return c.Next() },
	}
	plugin2 := &mockGlobalPlugin{
		name:    "nil-handler",
		handler: nil,
	}
	plugin3 := &mockGlobalPlugin{
		name:    "valid2",
		handler: func(c *fiber.Ctx) error { return c.Next() },
	}

	registry.RegisterGlobal(plugin1)
	registry.RegisterGlobal(plugin2)
	registry.RegisterGlobal(plugin3)

	err := registry.ApplyGlobal(app)

	if err == nil {
		t.Fatal("expected error for nil handler")
	}

	if err.Error() != "global plugin 'nil-handler' returned nil handler" {
		t.Errorf("expected error about nil-handler, got '%s'", err.Error())
	}
}

func TestApplyGlobal_PluginOrder(t *testing.T) {
	registry := NewPluginRegistry()
	app := fiber.New()

	plugin1 := &mockGlobalPlugin{
		name: "first",
		handler: func(c *fiber.Ctx) error {
			return c.Next()
		},
	}
	plugin2 := &mockGlobalPlugin{
		name: "second",
		handler: func(c *fiber.Ctx) error {
			return c.Next()
		},
	}
	plugin3 := &mockGlobalPlugin{
		name: "third",
		handler: func(c *fiber.Ctx) error {
			return c.Next()
		},
	}

	registry.RegisterGlobal(plugin1)
	registry.RegisterGlobal(plugin2)
	registry.RegisterGlobal(plugin3)

	err := registry.ApplyGlobal(app)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPluginRegistry_IntegrationScenario(t *testing.T) {
	registry := NewPluginRegistry()

	globalPlugin1 := &mockGlobalPlugin{
		name:    "logger",
		handler: func(c *fiber.Ctx) error { return c.Next() },
	}
	globalPlugin2 := &mockGlobalPlugin{
		name:    "cors",
		handler: func(c *fiber.Ctx) error { return c.Next() },
	}

	routePlugin1 := &mockRoutePlugin{name: "auth"}
	routePlugin2 := &mockRoutePlugin{name: "rate-limiter"}

	registry.RegisterGlobal(globalPlugin1)
	registry.RegisterGlobal(globalPlugin2)
	registry.RegisterRoute(routePlugin1)
	registry.RegisterRoute(routePlugin2)

	if len(registry.GetGlobalPlugins()) != 2 {
		t.Errorf("expected 2 global plugins, got %d", len(registry.GetGlobalPlugins()))
	}

	if len(registry.GetRoutePlugins()) != 2 {
		t.Errorf("expected 2 route plugins, got %d", len(registry.GetRoutePlugins()))
	}

	if plugin, exists := registry.GetRoutePlugin("auth"); !exists || plugin.Name() != "auth" {
		t.Error("expected auth plugin to exist")
	}

	if plugin, exists := registry.GetRoutePlugin("rate-limiter"); !exists || plugin.Name() != "rate-limiter" {
		t.Error("expected rate-limiter plugin to exist")
	}

	app := fiber.New()
	if err := registry.ApplyGlobal(app); err != nil {
		t.Errorf("expected no error applying global plugins, got %v", err)
	}
}

package pluginloader

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/database"
	"github.com/nicolasbonnici/gorest/plugin"
)

type mockGlobalPlugin struct {
	name        string
	initErr     error
	initCalled  bool
	config      map[string]interface{}
	handlerFunc fiber.Handler
}

func (m *mockGlobalPlugin) Name() string { return m.name }

func (m *mockGlobalPlugin) Initialize(cfg map[string]interface{}) error {
	m.initCalled = true
	m.config = cfg
	return m.initErr
}

func (m *mockGlobalPlugin) Handler() fiber.Handler {
	if m.handlerFunc != nil {
		return m.handlerFunc
	}
	return func(c *fiber.Ctx) error { return c.Next() }
}

type mockRoutePlugin struct {
	name       string
	initErr    error
	initCalled bool
	config     map[string]interface{}
}

func (m *mockRoutePlugin) Name() string { return m.name }

func (m *mockRoutePlugin) Initialize(cfg map[string]interface{}) error {
	m.initCalled = true
	m.config = cfg
	return m.initErr
}

func (m *mockRoutePlugin) Wrap(handler fiber.Handler) fiber.Handler {
	return handler
}

type mockDatabase struct{}

func (m *mockDatabase) Connect(ctx context.Context, dsn string) error                                        { return nil }
func (m *mockDatabase) Close() error                                                                         { return nil }
func (m *mockDatabase) Ping(ctx context.Context) error                                                       { return nil }
func (m *mockDatabase) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) { return nil, nil }
func (m *mockDatabase) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row         { return nil }
func (m *mockDatabase) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) { return nil, nil }
func (m *mockDatabase) Begin(ctx context.Context) (database.Tx, error)                                       { return nil, nil }
func (m *mockDatabase) Dialect() database.Dialect                                                            { return &mockDialect{} }
func (m *mockDatabase) DriverName() string                                                                    { return "mock" }
func (m *mockDatabase) Introspector() database.SchemaIntrospector                                             { return nil }

type mockDialect struct{}

func (d *mockDialect) Placeholder(n int) string                  { return "?" }
func (d *mockDialect) SupportsReturning() bool                   { return false }
func (d *mockDialect) ReturningClause(cols ...string) string     { return "" }
func (d *mockDialect) LimitOffset(limit, offset int) string      { return "" }
func (d *mockDialect) QuoteIdentifier(name string) string        { return name }
func (d *mockDialect) MapType(dbType string) string              { return dbType }
func (d *mockDialect) CaseInsensitiveLike() string               { return "LOWER" }

func TestLoadPlugins_Success(t *testing.T) {
	globalPluginFactories = make(map[string]GlobalPluginFactory)
	routePluginFactories = make(map[string]RoutePluginFactory)

	RegisterGlobalPluginFactory("test-global", func() plugin.GlobalPlugin {
		return &mockGlobalPlugin{
			name:        "test-global",
			handlerFunc: func(c *fiber.Ctx) error { return c.Next() },
		}
	})

	RegisterRoutePluginFactory("test-route", func() plugin.RoutePlugin {
		return &mockRoutePlugin{name: "test-route"}
	})

	globalConfigs := []config.PluginConfig{
		{Name: "test-global", Enabled: true, Config: map[string]interface{}{"key": "value"}},
	}

	routeConfigs := []config.PluginConfig{
		{Name: "test-route", Enabled: true, Config: map[string]interface{}{}},
	}

	registry, err := LoadPlugins(globalConfigs, routeConfigs, "1.0.0")
	if err != nil {
		t.Fatalf("LoadPlugins() failed: %v", err)
	}

	if len(registry.GetGlobalPlugins()) != 1 {
		t.Errorf("Expected 1 global plugin, got %d", len(registry.GetGlobalPlugins()))
	}

	if len(registry.GetRoutePlugins()) != 1 {
		t.Errorf("Expected 1 route plugin, got %d", len(registry.GetRoutePlugins()))
	}
}

func TestLoadPlugins_UnknownGlobalPlugin(t *testing.T) {
	globalPluginFactories = make(map[string]GlobalPluginFactory)

	globalConfigs := []config.PluginConfig{
		{Name: "unknown-plugin", Enabled: true},
	}

	_, err := LoadPlugins(globalConfigs, nil, "1.0.0")
	if err == nil {
		t.Fatal("Expected error for unknown plugin")
	}

	if !strings.Contains(err.Error(), "unknown-plugin") {
		t.Errorf("Error should mention plugin name, got: %v", err)
	}

	if !strings.Contains(err.Error(), "did you forget to register") {
		t.Errorf("Error should provide helpful hint, got: %v", err)
	}
}

func TestLoadPlugins_UnknownRoutePlugin(t *testing.T) {
	routePluginFactories = make(map[string]RoutePluginFactory)

	routeConfigs := []config.PluginConfig{
		{Name: "unknown-route", Enabled: true},
	}

	_, err := LoadPlugins(nil, routeConfigs, "1.0.0")
	if err == nil {
		t.Fatal("Expected error for unknown route plugin")
	}

	if !strings.Contains(err.Error(), "unknown-route") {
		t.Errorf("Error should mention plugin name, got: %v", err)
	}
}

func TestLoadPlugins_GlobalInitializationFailure(t *testing.T) {
	globalPluginFactories = make(map[string]GlobalPluginFactory)

	RegisterGlobalPluginFactory("failing-plugin", func() plugin.GlobalPlugin {
		return &mockGlobalPlugin{
			name:    "failing-plugin",
			initErr: errors.New("initialization failed"),
		}
	})

	globalConfigs := []config.PluginConfig{
		{Name: "failing-plugin", Enabled: true},
	}

	_, err := LoadPlugins(globalConfigs, nil, "1.0.0")
	if err == nil {
		t.Fatal("Expected error for plugin initialization failure")
	}

	if !strings.Contains(err.Error(), "failed to initialize") {
		t.Errorf("Error should indicate initialization failure, got: %v", err)
	}

	if !strings.Contains(err.Error(), "failing-plugin") {
		t.Errorf("Error should mention plugin name, got: %v", err)
	}
}

func TestLoadPlugins_RouteInitializationFailure(t *testing.T) {
	routePluginFactories = make(map[string]RoutePluginFactory)

	RegisterRoutePluginFactory("failing-route", func() plugin.RoutePlugin {
		return &mockRoutePlugin{
			name:    "failing-route",
			initErr: errors.New("route init failed"),
		}
	})

	routeConfigs := []config.PluginConfig{
		{Name: "failing-route", Enabled: true},
	}

	_, err := LoadPlugins(nil, routeConfigs, "1.0.0")
	if err == nil {
		t.Fatal("Expected error for route plugin initialization failure")
	}

	if !strings.Contains(err.Error(), "failed to initialize") {
		t.Errorf("Error should indicate initialization failure, got: %v", err)
	}
}

func TestLoadPlugins_SkipsDisabled(t *testing.T) {
	globalPluginFactories = make(map[string]GlobalPluginFactory)

	RegisterGlobalPluginFactory("test-global", func() plugin.GlobalPlugin {
		return &mockGlobalPlugin{name: "test-global"}
	})

	globalConfigs := []config.PluginConfig{
		{Name: "test-global", Enabled: false},
	}

	registry, err := LoadPlugins(globalConfigs, nil, "1.0.0")
	if err != nil {
		t.Fatalf("LoadPlugins() failed: %v", err)
	}

	if len(registry.GetGlobalPlugins()) != 0 {
		t.Errorf("Disabled plugins should not be loaded, got %d", len(registry.GetGlobalPlugins()))
	}
}

func TestLoadPlugins_InjectsVersion(t *testing.T) {
	globalPluginFactories = make(map[string]GlobalPluginFactory)

	mockPlugin := &mockGlobalPlugin{name: "version-plugin"}

	RegisterGlobalPluginFactory("version-plugin", func() plugin.GlobalPlugin {
		return mockPlugin
	})

	globalConfigs := []config.PluginConfig{
		{Name: "version-plugin", Enabled: true, Config: map[string]interface{}{"custom": "value"}},
	}

	_, err := LoadPlugins(globalConfigs, nil, "2.5.0")
	if err != nil {
		t.Fatalf("LoadPlugins() failed: %v", err)
	}

	if !mockPlugin.initCalled {
		t.Fatal("Plugin Initialize() should have been called")
	}

	if version, ok := mockPlugin.config["__version"].(string); !ok || version != "2.5.0" {
		t.Errorf("Version should be injected into plugin config, got: %v", mockPlugin.config["__version"])
	}

	if custom, ok := mockPlugin.config["custom"].(string); !ok || custom != "value" {
		t.Errorf("Original config should be preserved, got: %v", mockPlugin.config["custom"])
	}
}

func TestRegisterGlobalPluginFactory(t *testing.T) {
	globalPluginFactories = make(map[string]GlobalPluginFactory)

	factory := func() plugin.GlobalPlugin {
		return &mockGlobalPlugin{name: "test"}
	}

	RegisterGlobalPluginFactory("test", factory)

	if len(globalPluginFactories) != 1 {
		t.Errorf("Expected 1 factory registered, got %d", len(globalPluginFactories))
	}

	if _, exists := globalPluginFactories["test"]; !exists {
		t.Error("Factory should be registered with name 'test'")
	}
}

func TestRegisterRoutePluginFactory(t *testing.T) {
	routePluginFactories = make(map[string]RoutePluginFactory)

	factory := func() plugin.RoutePlugin {
		return &mockRoutePlugin{name: "test-route"}
	}

	RegisterRoutePluginFactory("test-route", factory)

	if len(routePluginFactories) != 1 {
		t.Errorf("Expected 1 factory registered, got %d", len(routePluginFactories))
	}

	if _, exists := routePluginFactories["test-route"]; !exists {
		t.Error("Factory should be registered with name 'test-route'")
	}
}

func TestInjectSharedConfig(t *testing.T) {
	mockDB := &mockDatabase{}

	routeConfigs := []config.PluginConfig{
		{
			Name:    "auth",
			Enabled: true,
			Config: map[string]interface{}{
				"jwt_secret": "secret123",
				"jwt_ttl":    900,
			},
		},
		{
			Name:    "custom",
			Enabled: true,
			Config: map[string]interface{}{
				"api_key": "key456",
			},
		},
	}

	enriched := InjectSharedConfig(routeConfigs, mockDB)

	if len(enriched) != 2 {
		t.Fatalf("Expected 2 enriched configs, got %d", len(enriched))
	}

	if _, ok := enriched[0].Config["database"]; !ok {
		t.Error("Database should be injected into first config")
	}

	if _, ok := enriched[1].Config["database"]; !ok {
		t.Error("Database should be injected into second config")
	}

	if enriched[0].Config["jwt_secret"] != "secret123" {
		t.Error("Original config values should be preserved")
	}

	if enriched[0].Config["jwt_ttl"] != 900 {
		t.Error("Original config values should be preserved")
	}

	if enriched[1].Config["api_key"] != "key456" {
		t.Error("Original config values should be preserved")
	}
}

func TestInjectSharedConfig_EmptyConfig(t *testing.T) {
	mockDB := &mockDatabase{}

	routeConfigs := []config.PluginConfig{
		{
			Name:    "simple",
			Enabled: true,
			Config:  map[string]interface{}{},
		},
	}

	enriched := InjectSharedConfig(routeConfigs, mockDB)

	if len(enriched) != 1 {
		t.Fatalf("Expected 1 enriched config, got %d", len(enriched))
	}

	if _, ok := enriched[0].Config["database"]; !ok {
		t.Error("Database should be injected even with empty config")
	}
}

func TestSetupPluginEndpoints_NoPlugins(t *testing.T) {
	registry := plugin.NewPluginRegistry()
	app := fiber.New()

	err := SetupPluginEndpoints(registry, app)
	if err != nil {
		t.Errorf("Should not error with no plugins, got: %v", err)
	}
}

func TestSetupPluginEndpoints_NoEndpointSetupInterface(t *testing.T) {
	globalPluginFactories = make(map[string]GlobalPluginFactory)

	RegisterGlobalPluginFactory("simple", func() plugin.GlobalPlugin {
		return &mockGlobalPlugin{name: "simple"}
	})

	globalConfigs := []config.PluginConfig{
		{Name: "simple", Enabled: true, Config: map[string]interface{}{}},
	}

	registry, _ := LoadPlugins(globalConfigs, nil, "1.0.0")
	app := fiber.New()

	err := SetupPluginEndpoints(registry, app)
	if err != nil {
		t.Errorf("Should not error when plugins don't implement EndpointSetup, got: %v", err)
	}
}

func TestLoadPlugins_MultiplePluginsOrder(t *testing.T) {
	globalPluginFactories = make(map[string]GlobalPluginFactory)

	RegisterGlobalPluginFactory("plugin1", func() plugin.GlobalPlugin {
		return &mockGlobalPlugin{name: "plugin1"}
	})
	RegisterGlobalPluginFactory("plugin2", func() plugin.GlobalPlugin {
		return &mockGlobalPlugin{name: "plugin2"}
	})
	RegisterGlobalPluginFactory("plugin3", func() plugin.GlobalPlugin {
		return &mockGlobalPlugin{name: "plugin3"}
	})

	globalConfigs := []config.PluginConfig{
		{Name: "plugin1", Enabled: true, Config: map[string]interface{}{}},
		{Name: "plugin2", Enabled: true, Config: map[string]interface{}{}},
		{Name: "plugin3", Enabled: true, Config: map[string]interface{}{}},
	}

	registry, err := LoadPlugins(globalConfigs, nil, "1.0.0")
	if err != nil {
		t.Fatalf("LoadPlugins() failed: %v", err)
	}

	plugins := registry.GetGlobalPlugins()
	if len(plugins) != 3 {
		t.Fatalf("Expected 3 plugins, got %d", len(plugins))
	}

	if plugins[0].Name() != "plugin1" {
		t.Error("Plugin order should be preserved (plugin1)")
	}
	if plugins[1].Name() != "plugin2" {
		t.Error("Plugin order should be preserved (plugin2)")
	}
	if plugins[2].Name() != "plugin3" {
		t.Error("Plugin order should be preserved (plugin3)")
	}
}

func TestLoadPlugins_MixedEnabledDisabled(t *testing.T) {
	globalPluginFactories = make(map[string]GlobalPluginFactory)

	RegisterGlobalPluginFactory("enabled1", func() plugin.GlobalPlugin {
		return &mockGlobalPlugin{name: "enabled1"}
	})
	RegisterGlobalPluginFactory("disabled", func() plugin.GlobalPlugin {
		return &mockGlobalPlugin{name: "disabled"}
	})
	RegisterGlobalPluginFactory("enabled2", func() plugin.GlobalPlugin {
		return &mockGlobalPlugin{name: "enabled2"}
	})

	globalConfigs := []config.PluginConfig{
		{Name: "enabled1", Enabled: true, Config: map[string]interface{}{}},
		{Name: "disabled", Enabled: false, Config: map[string]interface{}{}},
		{Name: "enabled2", Enabled: true, Config: map[string]interface{}{}},
	}

	registry, err := LoadPlugins(globalConfigs, nil, "1.0.0")
	if err != nil {
		t.Fatalf("LoadPlugins() failed: %v", err)
	}

	plugins := registry.GetGlobalPlugins()
	if len(plugins) != 2 {
		t.Fatalf("Expected 2 enabled plugins, got %d", len(plugins))
	}

	if plugins[0].Name() != "enabled1" {
		t.Error("First plugin should be enabled1")
	}
	if plugins[1].Name() != "enabled2" {
		t.Error("Second plugin should be enabled2")
	}
}

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

type mockPlugin struct {
	name        string
	initErr     error
	initCalled  bool
	config      map[string]interface{}
	handlerFunc fiber.Handler
}

func (m *mockPlugin) Name() string { return m.name }

func (m *mockPlugin) Initialize(cfg map[string]interface{}) error {
	m.initCalled = true
	m.config = cfg
	return m.initErr
}

func (m *mockPlugin) Handler() fiber.Handler {
	if m.handlerFunc != nil {
		return m.handlerFunc
	}
	return func(c *fiber.Ctx) error { return c.Next() }
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
	pluginFactories = make(map[string]PluginFactory)

	RegisterPluginFactory("test-plugin1", func() plugin.Plugin {
		return &mockPlugin{
			name:        "test-plugin1",
			handlerFunc: func(c *fiber.Ctx) error { return c.Next() },
		}
	})

	RegisterPluginFactory("test-plugin2", func() plugin.Plugin {
		return &mockPlugin{name: "test-plugin2"}
	})

	configs := []config.PluginConfig{
		{Name: "test-plugin1", Enabled: true, Config: map[string]interface{}{"key": "value"}},
		{Name: "test-plugin2", Enabled: true, Config: map[string]interface{}{}},
	}

	registry, err := LoadPlugins(configs, "1.0.0")
	if err != nil {
		t.Fatalf("LoadPlugins() failed: %v", err)
	}

	if len(registry.GetAll()) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(registry.GetAll()))
	}
}

func TestLoadPlugins_UnknownPlugin(t *testing.T) {
	pluginFactories = make(map[string]PluginFactory)

	configs := []config.PluginConfig{
		{Name: "unknown-plugin", Enabled: true},
	}

	_, err := LoadPlugins(configs, "1.0.0")
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

func TestLoadPlugins_InitializationFailure(t *testing.T) {
	pluginFactories = make(map[string]PluginFactory)

	RegisterPluginFactory("failing-plugin", func() plugin.Plugin {
		return &mockPlugin{
			name:    "failing-plugin",
			initErr: errors.New("initialization failed"),
		}
	})

	configs := []config.PluginConfig{
		{Name: "failing-plugin", Enabled: true},
	}

	_, err := LoadPlugins(configs, "1.0.0")
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

func TestLoadPlugins_SkipsDisabled(t *testing.T) {
	pluginFactories = make(map[string]PluginFactory)

	RegisterPluginFactory("test-plugin", func() plugin.Plugin {
		return &mockPlugin{name: "test-plugin"}
	})

	configs := []config.PluginConfig{
		{Name: "test-plugin", Enabled: false},
	}

	registry, err := LoadPlugins(configs, "1.0.0")
	if err != nil {
		t.Fatalf("LoadPlugins() failed: %v", err)
	}

	if len(registry.GetAll()) != 0 {
		t.Errorf("Disabled plugins should not be loaded, got %d", len(registry.GetAll()))
	}
}

func TestLoadPlugins_InjectsVersion(t *testing.T) {
	pluginFactories = make(map[string]PluginFactory)

	mockPluginInstance := &mockPlugin{name: "version-plugin"}

	RegisterPluginFactory("version-plugin", func() plugin.Plugin {
		return mockPluginInstance
	})

	configs := []config.PluginConfig{
		{Name: "version-plugin", Enabled: true, Config: map[string]interface{}{"custom": "value"}},
	}

	_, err := LoadPlugins(configs, "2.5.0")
	if err != nil {
		t.Fatalf("LoadPlugins() failed: %v", err)
	}

	if !mockPluginInstance.initCalled {
		t.Fatal("Plugin Initialize() should have been called")
	}

	if version, ok := mockPluginInstance.config["__version"].(string); !ok || version != "2.5.0" {
		t.Errorf("Version should be injected into plugin config, got: %v", mockPluginInstance.config["__version"])
	}

	if custom, ok := mockPluginInstance.config["custom"].(string); !ok || custom != "value" {
		t.Errorf("Original config should be preserved, got: %v", mockPluginInstance.config["custom"])
	}
}

func TestRegisterPluginFactory(t *testing.T) {
	pluginFactories = make(map[string]PluginFactory)

	factory := func() plugin.Plugin {
		return &mockPlugin{name: "test"}
	}

	RegisterPluginFactory("test", factory)

	if len(pluginFactories) != 1 {
		t.Errorf("Expected 1 factory registered, got %d", len(pluginFactories))
	}

	if _, exists := pluginFactories["test"]; !exists {
		t.Error("Factory should be registered with name 'test'")
	}
}

func TestInjectSharedConfig(t *testing.T) {
	mockDB := &mockDatabase{}

	configs := []config.PluginConfig{
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

	appConfig := &config.Config{}
	enriched := InjectSharedConfig(configs, mockDB, appConfig)

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

	configs := []config.PluginConfig{
		{
			Name:    "simple",
			Enabled: true,
			Config:  map[string]interface{}{},
		},
	}

	appConfig := &config.Config{}
	enriched := InjectSharedConfig(configs, mockDB, appConfig)

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
	pluginFactories = make(map[string]PluginFactory)

	RegisterPluginFactory("simple", func() plugin.Plugin {
		return &mockPlugin{name: "simple"}
	})

	configs := []config.PluginConfig{
		{Name: "simple", Enabled: true, Config: map[string]interface{}{}},
	}

	registry, _ := LoadPlugins(configs, "1.0.0")
	app := fiber.New()

	err := SetupPluginEndpoints(registry, app)
	if err != nil {
		t.Errorf("Should not error when plugins don't implement EndpointSetup, got: %v", err)
	}
}

func TestLoadPlugins_MultiplePluginsOrder(t *testing.T) {
	pluginFactories = make(map[string]PluginFactory)

	RegisterPluginFactory("plugin1", func() plugin.Plugin {
		return &mockPlugin{name: "plugin1"}
	})
	RegisterPluginFactory("plugin2", func() plugin.Plugin {
		return &mockPlugin{name: "plugin2"}
	})
	RegisterPluginFactory("plugin3", func() plugin.Plugin {
		return &mockPlugin{name: "plugin3"}
	})

	configs := []config.PluginConfig{
		{Name: "plugin1", Enabled: true, Config: map[string]interface{}{}},
		{Name: "plugin2", Enabled: true, Config: map[string]interface{}{}},
		{Name: "plugin3", Enabled: true, Config: map[string]interface{}{}},
	}

	registry, err := LoadPlugins(configs, "1.0.0")
	if err != nil {
		t.Fatalf("LoadPlugins() failed: %v", err)
	}

	plugins := registry.GetAll()
	if len(plugins) != 3 {
		t.Fatalf("Expected 3 plugins, got %d", len(plugins))
	}

	// Check all plugins exist (order isn't guaranteed in map)
	if _, exists := plugins["plugin1"]; !exists {
		t.Error("Expected plugin1 to exist")
	}
	if _, exists := plugins["plugin2"]; !exists {
		t.Error("Expected plugin2 to exist")
	}
	if _, exists := plugins["plugin3"]; !exists {
		t.Error("Expected plugin3 to exist")
	}
}

func TestLoadPlugins_MixedEnabledDisabled(t *testing.T) {
	pluginFactories = make(map[string]PluginFactory)

	RegisterPluginFactory("enabled1", func() plugin.Plugin {
		return &mockPlugin{name: "enabled1"}
	})
	RegisterPluginFactory("disabled", func() plugin.Plugin {
		return &mockPlugin{name: "disabled"}
	})
	RegisterPluginFactory("enabled2", func() plugin.Plugin {
		return &mockPlugin{name: "enabled2"}
	})

	configs := []config.PluginConfig{
		{Name: "enabled1", Enabled: true, Config: map[string]interface{}{}},
		{Name: "disabled", Enabled: false, Config: map[string]interface{}{}},
		{Name: "enabled2", Enabled: true, Config: map[string]interface{}{}},
	}

	registry, err := LoadPlugins(configs, "1.0.0")
	if err != nil {
		t.Fatalf("LoadPlugins() failed: %v", err)
	}

	plugins := registry.GetAll()
	if len(plugins) != 2 {
		t.Fatalf("Expected 2 enabled plugins, got %d", len(plugins))
	}

	if _, exists := plugins["enabled1"]; !exists {
		t.Error("Expected enabled1 to exist")
	}
	if _, exists := plugins["enabled2"]; !exists {
		t.Error("Expected enabled2 to exist")
	}
	if _, exists := plugins["disabled"]; exists {
		t.Error("Expected disabled plugin not to exist")
	}
}

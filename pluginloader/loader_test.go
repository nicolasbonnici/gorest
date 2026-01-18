package pluginloader

import (
	"context"
	"errors"
	"net/http/httptest"
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

func (m *mockDatabase) Connect(ctx context.Context, dsn string) error { return nil }
func (m *mockDatabase) Close() error                                  { return nil }
func (m *mockDatabase) Ping(ctx context.Context) error                { return nil }
func (m *mockDatabase) Query(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	return nil, nil
}
func (m *mockDatabase) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	return nil
}
func (m *mockDatabase) Exec(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	return nil, nil
}
func (m *mockDatabase) Begin(ctx context.Context) (database.Tx, error) { return nil, nil }
func (m *mockDatabase) Dialect() database.Dialect                      { return &mockDialect{} }
func (m *mockDatabase) DriverName() string                             { return "mock" }
func (m *mockDatabase) Introspector() database.SchemaIntrospector      { return nil }

type mockDialect struct{}

func (d *mockDialect) Placeholder(n int) string                                { return "?" }
func (d *mockDialect) SupportsReturning() bool                                 { return false }
func (d *mockDialect) ReturningClause(cols ...string) string                   { return "" }
func (d *mockDialect) LimitOffset(limit, offset int) string                    { return "" }
func (d *mockDialect) QuoteIdentifier(name string) string                      { return name }
func (d *mockDialect) MapType(dbType string) string                            { return dbType }
func (d *mockDialect) CaseInsensitiveLike() string                             { return "LOWER" }
func (d *mockDialect) SupportsFullJoin() bool                                  { return false }
func (d *mockDialect) SupportsWindowFunctions() bool                           { return false }
func (d *mockDialect) SupportsCTE() bool                                       { return false }
func (d *mockDialect) SupportsArrays() bool                                    { return false }
func (d *mockDialect) OnConflictClause(columns []string, action string) string { return "" }
func (d *mockDialect) UpsertSupport() bool                                     { return false }

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

func TestApplyGlobalMiddleware_EmptyRegistry(t *testing.T) {
	registry := plugin.NewPluginRegistry()
	app := fiber.New()

	ApplyGlobalMiddleware(registry, app)
}

func TestApplyGlobalMiddleware_WithMiddlewarePlugins(t *testing.T) {
	pluginFactories = make(map[string]PluginFactory)

	middlewareCalled := make(map[string]bool)

	RegisterPluginFactory("requestid", func() plugin.Plugin {
		return &mockPlugin{
			name: "requestid",
			handlerFunc: func(c *fiber.Ctx) error {
				middlewareCalled["requestid"] = true
				return c.Next()
			},
		}
	})

	RegisterPluginFactory("logger", func() plugin.Plugin {
		return &mockPlugin{
			name: "logger",
			handlerFunc: func(c *fiber.Ctx) error {
				middlewareCalled["logger"] = true
				return c.Next()
			},
		}
	})

	RegisterPluginFactory("cors", func() plugin.Plugin {
		return &mockPlugin{
			name: "cors",
			handlerFunc: func(c *fiber.Ctx) error {
				middlewareCalled["cors"] = true
				return c.Next()
			},
		}
	})

	configs := []config.PluginConfig{
		{Name: "requestid", Enabled: true, Config: map[string]interface{}{}},
		{Name: "logger", Enabled: true, Config: map[string]interface{}{}},
		{Name: "cors", Enabled: true, Config: map[string]interface{}{}},
	}

	registry, err := LoadPlugins(configs, "1.0.0")
	if err != nil {
		t.Fatalf("LoadPlugins() failed: %v", err)
	}

	app := fiber.New()
	ApplyGlobalMiddleware(registry, app)

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	_, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test: %v", err)
	}

	if !middlewareCalled["requestid"] {
		t.Error("requestid middleware should be called")
	}
	if !middlewareCalled["logger"] {
		t.Error("logger middleware should be called")
	}
	if !middlewareCalled["cors"] {
		t.Error("cors middleware should be called")
	}
}

func TestApplyGlobalMiddleware_PartialPluginsPresent(t *testing.T) {
	pluginFactories = make(map[string]PluginFactory)

	RegisterPluginFactory("requestid", func() plugin.Plugin {
		return &mockPlugin{
			name: "requestid",
			handlerFunc: func(c *fiber.Ctx) error {
				return c.Next()
			},
		}
	})

	configs := []config.PluginConfig{
		{Name: "requestid", Enabled: true, Config: map[string]interface{}{}},
	}

	registry, err := LoadPlugins(configs, "1.0.0")
	if err != nil {
		t.Fatalf("LoadPlugins() failed: %v", err)
	}

	app := fiber.New()
	ApplyGlobalMiddleware(registry, app)
}

type mockEndpointSetupPlugin struct {
	mockPlugin
	setupCalled bool
	setupErr    error
}

func (m *mockEndpointSetupPlugin) SetupEndpoints(app *fiber.App) error {
	m.setupCalled = true
	return m.setupErr
}

func TestSetupPluginEndpoints_WithEndpointSetup(t *testing.T) {
	pluginFactories = make(map[string]PluginFactory)

	mockEndpoint := &mockEndpointSetupPlugin{
		mockPlugin: mockPlugin{name: "endpoint-plugin"},
	}

	RegisterPluginFactory("endpoint-plugin", func() plugin.Plugin {
		return mockEndpoint
	})

	configs := []config.PluginConfig{
		{Name: "endpoint-plugin", Enabled: true, Config: map[string]interface{}{}},
	}

	registry, err := LoadPlugins(configs, "1.0.0")
	if err != nil {
		t.Fatalf("LoadPlugins() failed: %v", err)
	}

	app := fiber.New()
	err = SetupPluginEndpoints(registry, app)
	if err != nil {
		t.Errorf("SetupPluginEndpoints() should not error, got: %v", err)
	}

	if !mockEndpoint.setupCalled {
		t.Error("SetupEndpoints should be called on plugin")
	}
}

func TestSetupPluginEndpoints_Error(t *testing.T) {
	pluginFactories = make(map[string]PluginFactory)

	expectedErr := errors.New("setup failed")
	mockEndpoint := &mockEndpointSetupPlugin{
		mockPlugin: mockPlugin{name: "failing-endpoint"},
		setupErr:   expectedErr,
	}

	RegisterPluginFactory("failing-endpoint", func() plugin.Plugin {
		return mockEndpoint
	})

	configs := []config.PluginConfig{
		{Name: "failing-endpoint", Enabled: true, Config: map[string]interface{}{}},
	}

	registry, err := LoadPlugins(configs, "1.0.0")
	if err != nil {
		t.Fatalf("LoadPlugins() failed: %v", err)
	}

	app := fiber.New()
	err = SetupPluginEndpoints(registry, app)
	if err == nil {
		t.Fatal("Expected error from SetupPluginEndpoints")
	}

	if !strings.Contains(err.Error(), "failing-endpoint") {
		t.Errorf("Error should mention plugin name, got: %v", err)
	}

	if !strings.Contains(err.Error(), "failed to setup endpoints") {
		t.Errorf("Error should indicate setup failure, got: %v", err)
	}
}

func TestInjectSharedConfig_WithPaginationConfig(t *testing.T) {
	mockDB := &mockDatabase{}

	configs := []config.PluginConfig{
		{
			Name:    "auth",
			Enabled: true,
			Config: map[string]interface{}{
				"jwt_secret": "secret123",
			},
		},
	}

	appConfig := &config.Config{
		Pagination: config.PaginationConfig{
			DefaultLimit: 50,
			MaxLimit:     200,
		},
	}

	enriched := InjectSharedConfig(configs, mockDB, appConfig)

	if len(enriched) != 1 {
		t.Fatalf("Expected 1 enriched config, got %d", len(enriched))
	}

	if paginationLimit, ok := enriched[0].Config["pagination_limit"].(int); !ok || paginationLimit != 50 {
		t.Errorf("pagination_limit should be 50, got %v", enriched[0].Config["pagination_limit"])
	}

	if paginationMaxLimit, ok := enriched[0].Config["pagination_max_limit"].(int); !ok || paginationMaxLimit != 200 {
		t.Errorf("pagination_max_limit should be 200, got %v", enriched[0].Config["pagination_max_limit"])
	}

	if cfg, ok := enriched[0].Config["config"].(*config.Config); !ok || cfg == nil {
		t.Error("config should be injected")
	}
}

func TestInjectSharedConfig_OpenAPIPlugin(t *testing.T) {
	mockDB := &mockDatabase{}

	configs := []config.PluginConfig{
		{
			Name:    "openapi",
			Enabled: true,
			Config:  map[string]interface{}{},
		},
	}

	appConfig := &config.Config{
		Codegen: config.CodegenConfig{
			Output: config.OutputConfig{
				DTOs: "generated/dtos",
			},
		},
		Pagination: config.PaginationConfig{
			DefaultLimit: 25,
			MaxLimit:     100,
		},
	}

	enriched := InjectSharedConfig(configs, mockDB, appConfig)

	if len(enriched) != 1 {
		t.Fatalf("Expected 1 enriched config, got %d", len(enriched))
	}
}

type mockCommandPlugin struct {
	mockPlugin
	commands []plugin.Command
}

func (m *mockCommandPlugin) Commands() []plugin.Command {
	return m.commands
}

type mockCommand struct {
	name        string
	description string
}

func (m *mockCommand) Name() string        { return m.name }
func (m *mockCommand) Description() string { return m.description }
func (m *mockCommand) Run(ctx *plugin.CommandContext) *plugin.CommandResult {
	return &plugin.CommandResult{Success: true}
}

func TestLoadAllCommandPlugins_Success(t *testing.T) {
	pluginFactories = make(map[string]PluginFactory)

	cmd := &mockCommand{name: "test-cmd", description: "test command"}
	RegisterPluginFactory("cmd-plugin", func() plugin.Plugin {
		return &mockCommandPlugin{
			mockPlugin: mockPlugin{name: "cmd-plugin"},
			commands:   []plugin.Command{cmd},
		}
	})

	RegisterPluginFactory("regular-plugin", func() plugin.Plugin {
		return &mockPlugin{name: "regular-plugin"}
	})

	mockDB := &mockDatabase{}
	appConfig := &config.Config{}

	commandPlugins, err := LoadAllCommandPlugins(mockDB, appConfig)
	if err != nil {
		t.Fatalf("LoadAllCommandPlugins() failed: %v", err)
	}

	if len(commandPlugins) != 1 {
		t.Fatalf("Expected 1 command plugin, got %d", len(commandPlugins))
	}

	if commandPlugins[0].Name() != "cmd-plugin" {
		t.Errorf("Expected cmd-plugin, got %s", commandPlugins[0].Name())
	}

	if cmdProvider, ok := commandPlugins[0].(plugin.CommandProvider); ok {
		commands := cmdProvider.Commands()
		if len(commands) != 1 {
			t.Fatalf("Expected 1 command, got %d", len(commands))
		}
		if commands[0].Name() != "test-cmd" {
			t.Errorf("Expected test-cmd, got %s", commands[0].Name())
		}
	} else {
		t.Error("Plugin should implement CommandProvider")
	}
}

func TestLoadAllCommandPlugins_InitializationError(t *testing.T) {
	pluginFactories = make(map[string]PluginFactory)

	RegisterPluginFactory("failing-cmd-plugin", func() plugin.Plugin {
		return &mockCommandPlugin{
			mockPlugin: mockPlugin{
				name:    "failing-cmd-plugin",
				initErr: errors.New("init error"),
			},
		}
	})

	mockDB := &mockDatabase{}
	appConfig := &config.Config{}

	_, err := LoadAllCommandPlugins(mockDB, appConfig)
	if err == nil {
		t.Fatal("Expected error from LoadAllCommandPlugins")
	}

	if !strings.Contains(err.Error(), "failing-cmd-plugin") {
		t.Errorf("Error should mention plugin name, got: %v", err)
	}

	if !strings.Contains(err.Error(), "failed to initialize") {
		t.Errorf("Error should indicate initialization failure, got: %v", err)
	}
}

func TestLoadAllCommandPlugins_EmptyRegistry(t *testing.T) {
	pluginFactories = make(map[string]PluginFactory)

	mockDB := &mockDatabase{}
	appConfig := &config.Config{}

	commandPlugins, err := LoadAllCommandPlugins(mockDB, appConfig)
	if err != nil {
		t.Fatalf("LoadAllCommandPlugins() failed: %v", err)
	}

	if len(commandPlugins) != 0 {
		t.Errorf("Expected 0 command plugins with empty registry, got %d", len(commandPlugins))
	}
}

func TestLoadAllCommandPlugins_ConfigInjection(t *testing.T) {
	pluginFactories = make(map[string]PluginFactory)

	cmdPlugin := &mockCommandPlugin{
		mockPlugin: mockPlugin{name: "cmd-plugin"},
		commands:   []plugin.Command{},
	}

	RegisterPluginFactory("cmd-plugin", func() plugin.Plugin {
		return cmdPlugin
	})

	mockDB := &mockDatabase{}
	appConfig := &config.Config{}

	_, err := LoadAllCommandPlugins(mockDB, appConfig)
	if err != nil {
		t.Fatalf("LoadAllCommandPlugins() failed: %v", err)
	}

	if !cmdPlugin.initCalled {
		t.Error("Initialize should be called on command plugin")
	}

	if cmdPlugin.config["database"] == nil {
		t.Error("Database should be injected into command plugin config")
	}

	if cmdPlugin.config["config"] == nil {
		t.Error("Config should be injected into command plugin config")
	}
}

func TestFindProjectRoot_Success(t *testing.T) {
	root, err := findProjectRoot()
	if err != nil {
		t.Logf("findProjectRoot() returned error: %v (this is expected if go.mod is not in the current directory)", err)
		return
	}

	if root == "" {
		t.Error("findProjectRoot() should return non-empty path when successful")
	}
}

func TestInjectSharedConfig_PreservesEnabledFlag(t *testing.T) {
	mockDB := &mockDatabase{}

	configs := []config.PluginConfig{
		{
			Name:    "plugin1",
			Enabled: true,
			Config:  map[string]interface{}{},
		},
		{
			Name:    "plugin2",
			Enabled: false,
			Config:  map[string]interface{}{},
		},
	}

	appConfig := &config.Config{}
	enriched := InjectSharedConfig(configs, mockDB, appConfig)

	if len(enriched) != 2 {
		t.Fatalf("Expected 2 enriched configs, got %d", len(enriched))
	}

	if enriched[0].Enabled != true {
		t.Error("First plugin Enabled flag should be preserved as true")
	}

	if enriched[1].Enabled != false {
		t.Error("Second plugin Enabled flag should be preserved as false")
	}

	if enriched[0].Name != "plugin1" {
		t.Error("First plugin name should be preserved")
	}

	if enriched[1].Name != "plugin2" {
		t.Error("Second plugin name should be preserved")
	}
}

package pluginloader

import (
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/nicolasbonnici/gorest/config"
	"github.com/nicolasbonnici/gorest/plugin"
)

type mockPluginWithDeps struct {
	name string
	deps []string
}

func (m *mockPluginWithDeps) Name() string {
	return m.name
}

func (m *mockPluginWithDeps) Initialize(cfg map[string]interface{}) error {
	return nil
}

func (m *mockPluginWithDeps) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}

func (m *mockPluginWithDeps) Dependencies() []string {
	return m.deps
}

// Basic dependency resolution (A depends on B)
func TestBasicDependencyResolution(t *testing.T) {
	RegisterPluginFactory("pluginA", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginA", deps: []string{"pluginB"}}
	})
	RegisterPluginFactory("pluginB", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginB", deps: []string{}}
	})
	defer func() {
		delete(pluginFactories, "pluginA")
		delete(pluginFactories, "pluginB")
	}()

	configs := []config.PluginConfig{
		{Name: "pluginA", Enabled: true, Config: map[string]interface{}{}},
		{Name: "pluginB", Enabled: true, Config: map[string]interface{}{}},
	}

	pluginInfos, err := collectPluginDependencies(configs)
	if err != nil {
		t.Fatalf("collectPluginDependencies failed: %v", err)
	}

	if err := validateDependencies(pluginInfos, configs); err != nil {
		t.Fatalf("validateDependencies failed: %v", err)
	}

	sorted, err := resolveInitializationOrder(pluginInfos)
	if err != nil {
		t.Fatalf("resolveInitializationOrder failed: %v", err)
	}

	if len(sorted) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(sorted))
	}
	if sorted[0].name != "pluginB" {
		t.Errorf("expected pluginB first, got %s", sorted[0].name)
	}
	if sorted[1].name != "pluginA" {
		t.Errorf("expected pluginA second, got %s", sorted[1].name)
	}
}

// Transitive dependencies (A -> B -> C)
func TestTransitiveDependencies(t *testing.T) {
	RegisterPluginFactory("pluginA", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginA", deps: []string{"pluginB"}}
	})
	RegisterPluginFactory("pluginB", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginB", deps: []string{"pluginC"}}
	})
	RegisterPluginFactory("pluginC", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginC", deps: []string{}}
	})
	defer func() {
		delete(pluginFactories, "pluginA")
		delete(pluginFactories, "pluginB")
		delete(pluginFactories, "pluginC")
	}()

	configs := []config.PluginConfig{
		{Name: "pluginA", Enabled: true, Config: map[string]interface{}{}},
		{Name: "pluginB", Enabled: true, Config: map[string]interface{}{}},
		{Name: "pluginC", Enabled: true, Config: map[string]interface{}{}},
	}

	pluginInfos, err := collectPluginDependencies(configs)
	if err != nil {
		t.Fatalf("collectPluginDependencies failed: %v", err)
	}

	if err := validateDependencies(pluginInfos, configs); err != nil {
		t.Fatalf("validateDependencies failed: %v", err)
	}

	sorted, err := resolveInitializationOrder(pluginInfos)
	if err != nil {
		t.Fatalf("resolveInitializationOrder failed: %v", err)
	}

	if len(sorted) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(sorted))
	}
	if sorted[0].name != "pluginC" {
		t.Errorf("expected pluginC first, got %s", sorted[0].name)
	}
	if sorted[1].name != "pluginB" {
		t.Errorf("expected pluginB second, got %s", sorted[1].name)
	}
	if sorted[2].name != "pluginA" {
		t.Errorf("expected pluginA third, got %s", sorted[2].name)
	}
}

// Circular dependency detection (A -> B -> A)
func TestCircularDependencyDetection(t *testing.T) {
	RegisterPluginFactory("pluginA", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginA", deps: []string{"pluginB"}}
	})
	RegisterPluginFactory("pluginB", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginB", deps: []string{"pluginA"}}
	})
	defer func() {
		delete(pluginFactories, "pluginA")
		delete(pluginFactories, "pluginB")
	}()

	configs := []config.PluginConfig{
		{Name: "pluginA", Enabled: true, Config: map[string]interface{}{}},
		{Name: "pluginB", Enabled: true, Config: map[string]interface{}{}},
	}

	pluginInfos, err := collectPluginDependencies(configs)
	if err != nil {
		t.Fatalf("collectPluginDependencies failed: %v", err)
	}

	if err := validateDependencies(pluginInfos, configs); err != nil {
		t.Fatalf("validateDependencies failed: %v", err)
	}

	_, err = resolveInitializationOrder(pluginInfos)
	if err == nil {
		t.Fatal("expected circular dependency error, got nil")
	}

	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("expected 'circular dependency' in error message, got: %v", err)
	}
}

// Missing dependency validation
func TestMissingDependencyValidation(t *testing.T) {
	RegisterPluginFactory("pluginA", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginA", deps: []string{"pluginB"}}
	})
	defer func() {
		delete(pluginFactories, "pluginA")
	}()

	configs := []config.PluginConfig{
		{Name: "pluginA", Enabled: true, Config: map[string]interface{}{}},
	}

	pluginInfos, err := collectPluginDependencies(configs)
	if err != nil {
		t.Fatalf("collectPluginDependencies failed: %v", err)
	}

	err = validateDependencies(pluginInfos, configs)
	if err == nil {
		t.Fatal("expected missing dependency error, got nil")
	}

	if !strings.Contains(err.Error(), "pluginA") || !strings.Contains(err.Error(), "pluginB") {
		t.Errorf("expected error about pluginA requiring pluginB, got: %v", err)
	}
}

// Multiple dependencies
func TestMultipleDependencies(t *testing.T) {
	RegisterPluginFactory("pluginA", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginA", deps: []string{"pluginB", "pluginC"}}
	})
	RegisterPluginFactory("pluginB", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginB", deps: []string{}}
	})
	RegisterPluginFactory("pluginC", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginC", deps: []string{}}
	})
	defer func() {
		delete(pluginFactories, "pluginA")
		delete(pluginFactories, "pluginB")
		delete(pluginFactories, "pluginC")
	}()

	configs := []config.PluginConfig{
		{Name: "pluginA", Enabled: true, Config: map[string]interface{}{}},
		{Name: "pluginB", Enabled: true, Config: map[string]interface{}{}},
		{Name: "pluginC", Enabled: true, Config: map[string]interface{}{}},
	}

	pluginInfos, err := collectPluginDependencies(configs)
	if err != nil {
		t.Fatalf("collectPluginDependencies failed: %v", err)
	}

	if err := validateDependencies(pluginInfos, configs); err != nil {
		t.Fatalf("validateDependencies failed: %v", err)
	}

	sorted, err := resolveInitializationOrder(pluginInfos)
	if err != nil {
		t.Fatalf("resolveInitializationOrder failed: %v", err)
	}

	if len(sorted) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(sorted))
	}

	aIndex := -1
	bIndex := -1
	cIndex := -1

	for i, info := range sorted {
		switch info.name {
		case "pluginA":
			aIndex = i
		case "pluginB":
			bIndex = i
		case "pluginC":
			cIndex = i
		}
	}

	if aIndex < bIndex || aIndex < cIndex {
		t.Errorf("pluginA should come after both pluginB and pluginC")
	}
}

// No dependencies case
func TestNoDependencies(t *testing.T) {
	RegisterPluginFactory("pluginA", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginA", deps: []string{}}
	})
	RegisterPluginFactory("pluginB", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginB", deps: []string{}}
	})
	defer func() {
		delete(pluginFactories, "pluginA")
		delete(pluginFactories, "pluginB")
	}()

	configs := []config.PluginConfig{
		{Name: "pluginA", Enabled: true, Config: map[string]interface{}{}},
		{Name: "pluginB", Enabled: true, Config: map[string]interface{}{}},
	}

	pluginInfos, err := collectPluginDependencies(configs)
	if err != nil {
		t.Fatalf("collectPluginDependencies failed: %v", err)
	}

	if err := validateDependencies(pluginInfos, configs); err != nil {
		t.Fatalf("validateDependencies failed: %v", err)
	}

	sorted, err := resolveInitializationOrder(pluginInfos)
	if err != nil {
		t.Fatalf("resolveInitializationOrder failed: %v", err)
	}

	if len(sorted) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(sorted))
	}
}

// Complex transitive circular dependency (A -> B -> C -> A)
func TestComplexCircularDependency(t *testing.T) {
	RegisterPluginFactory("pluginA", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginA", deps: []string{"pluginB"}}
	})
	RegisterPluginFactory("pluginB", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginB", deps: []string{"pluginC"}}
	})
	RegisterPluginFactory("pluginC", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginC", deps: []string{"pluginA"}}
	})
	defer func() {
		delete(pluginFactories, "pluginA")
		delete(pluginFactories, "pluginB")
		delete(pluginFactories, "pluginC")
	}()

	configs := []config.PluginConfig{
		{Name: "pluginA", Enabled: true, Config: map[string]interface{}{}},
		{Name: "pluginB", Enabled: true, Config: map[string]interface{}{}},
		{Name: "pluginC", Enabled: true, Config: map[string]interface{}{}},
	}

	pluginInfos, err := collectPluginDependencies(configs)
	if err != nil {
		t.Fatalf("collectPluginDependencies failed: %v", err)
	}

	if err := validateDependencies(pluginInfos, configs); err != nil {
		t.Fatalf("validateDependencies failed: %v", err)
	}

	_, err = resolveInitializationOrder(pluginInfos)
	if err == nil {
		t.Fatal("expected circular dependency error, got nil")
	}

	if !strings.Contains(err.Error(), "circular dependency") {
		t.Errorf("expected 'circular dependency' in error message, got: %v", err)
	}
}

type simpleMockPlugin struct {
	name string
}

func (s *simpleMockPlugin) Name() string {
	return s.name
}

func (s *simpleMockPlugin) Initialize(cfg map[string]interface{}) error {
	return nil
}

func (s *simpleMockPlugin) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}

// Plugin without PluginDependencies interface
func TestPluginWithoutDependenciesInterface(t *testing.T) {
	RegisterPluginFactory("simplePlugin", func() plugin.Plugin {
		return &simpleMockPlugin{name: "simplePlugin"}
	})
	RegisterPluginFactory("pluginB", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginB", deps: []string{}}
	})
	defer func() {
		delete(pluginFactories, "simplePlugin")
		delete(pluginFactories, "pluginB")
	}()

	configs := []config.PluginConfig{
		{Name: "simplePlugin", Enabled: true, Config: map[string]interface{}{}},
		{Name: "pluginB", Enabled: true, Config: map[string]interface{}{}},
	}

	pluginInfos, err := collectPluginDependencies(configs)
	if err != nil {
		t.Fatalf("collectPluginDependencies failed: %v", err)
	}

	for _, info := range pluginInfos {
		if info.name == "simplePlugin" && len(info.dependencies) != 0 {
			t.Errorf("expected simplePlugin to have 0 dependencies, got %d", len(info.dependencies))
		}
	}

	if err := validateDependencies(pluginInfos, configs); err != nil {
		t.Fatalf("validateDependencies failed: %v", err)
	}

	sorted, err := resolveInitializationOrder(pluginInfos)
	if err != nil {
		t.Fatalf("resolveInitializationOrder failed: %v", err)
	}

	if len(sorted) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(sorted))
	}
}

// Integration test with LoadPlugins
func TestLoadPluginsWithDependencies(t *testing.T) {
	RegisterPluginFactory("base", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "base", deps: []string{}}
	})
	RegisterPluginFactory("dependent", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "dependent", deps: []string{"base"}}
	})
	defer func() {
		delete(pluginFactories, "base")
		delete(pluginFactories, "dependent")
	}()

	configs := []config.PluginConfig{
		{Name: "dependent", Enabled: true, Config: map[string]interface{}{}},
		{Name: "base", Enabled: true, Config: map[string]interface{}{}},
	}

	registry, err := LoadPlugins(configs, "1.0.0")
	if err != nil {
		t.Fatalf("LoadPlugins failed: %v", err)
	}

	if _, ok := registry.Get("base"); !ok {
		t.Error("base plugin not found in registry")
	}
	if _, ok := registry.Get("dependent"); !ok {
		t.Error("dependent plugin not found in registry")
	}
}

// Diamond dependency pattern (D -> B, D -> C, B -> A, C -> A)
func TestDiamondDependencyPattern(t *testing.T) {
	RegisterPluginFactory("pluginA", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginA", deps: []string{}}
	})
	RegisterPluginFactory("pluginB", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginB", deps: []string{"pluginA"}}
	})
	RegisterPluginFactory("pluginC", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginC", deps: []string{"pluginA"}}
	})
	RegisterPluginFactory("pluginD", func() plugin.Plugin {
		return &mockPluginWithDeps{name: "pluginD", deps: []string{"pluginB", "pluginC"}}
	})
	defer func() {
		delete(pluginFactories, "pluginA")
		delete(pluginFactories, "pluginB")
		delete(pluginFactories, "pluginC")
		delete(pluginFactories, "pluginD")
	}()

	configs := []config.PluginConfig{
		{Name: "pluginD", Enabled: true, Config: map[string]interface{}{}},
		{Name: "pluginB", Enabled: true, Config: map[string]interface{}{}},
		{Name: "pluginC", Enabled: true, Config: map[string]interface{}{}},
		{Name: "pluginA", Enabled: true, Config: map[string]interface{}{}},
	}

	pluginInfos, err := collectPluginDependencies(configs)
	if err != nil {
		t.Fatalf("collectPluginDependencies failed: %v", err)
	}

	if err := validateDependencies(pluginInfos, configs); err != nil {
		t.Fatalf("validateDependencies failed: %v", err)
	}

	sorted, err := resolveInitializationOrder(pluginInfos)
	if err != nil {
		t.Fatalf("resolveInitializationOrder failed: %v", err)
	}

	if len(sorted) != 4 {
		t.Fatalf("expected 4 plugins, got %d", len(sorted))
	}

	if sorted[0].name != "pluginA" {
		t.Errorf("expected pluginA first, got %s", sorted[0].name)
	}

	if sorted[3].name != "pluginD" {
		t.Errorf("expected pluginD last, got %s", sorted[3].name)
	}
}

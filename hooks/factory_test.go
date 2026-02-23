package hooks

import (
	"testing"
)

// Test model for generic hooks testing
type testModel struct {
	ID   string
	Name string
}

func TestNewHookFactory(t *testing.T) {
	factory := NewHookFactory()
	if factory == nil {
		t.Fatal("Expected non-nil factory")
	}
	if factory.registry == nil {
		t.Error("Expected initialized registry map")
	}
	if len(factory.registry) != 0 {
		t.Errorf("Expected empty registry, got %d items", len(factory.registry))
	}
}

func TestHookFactory_Register(t *testing.T) {
	tests := []struct {
		name         string
		resourceName string
		hooks        interface{}
		expectStored bool
	}{
		{
			name:         "simple registration",
			resourceName: "users",
			hooks:        &NoOpHooks[testModel]{},
			expectStored: true,
		},
		{
			name:         "string type registration",
			resourceName: "strings",
			hooks:        &NoOpHooks[string]{},
			expectStored: true,
		},
		{
			name:         "int type registration",
			resourceName: "numbers",
			hooks:        &NoOpHooks[int]{},
			expectStored: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewHookFactory()
			factory.Register(tt.resourceName, tt.hooks)
			_, exists := factory.GetHooks(tt.resourceName)
			if exists != tt.expectStored {
				t.Errorf("Expected exists=%v, got %v", tt.expectStored, exists)
			}
		})
	}
}

func TestHookFactory_RegisterOverwrite(t *testing.T) {
	factory := NewHookFactory()

	// Register first hooks
	firstHooks := &NoOpHooks[testModel]{}
	factory.Register("users", firstHooks)

	// Overwrite with new hooks
	secondHooks := &NoOpHooks[string]{}
	factory.Register("users", secondHooks)

	// Verify overwrite
	hooks, exists := factory.GetHooks("users")
	if !exists {
		t.Fatal("Expected hooks to exist")
	}

	// Should be the second hooks (different type)
	if _, ok := hooks.(*NoOpHooks[string]); !ok {
		t.Error("Expected hooks to be overwritten with string type")
	}
}

func TestHookFactory_GetHooks(t *testing.T) {
	tests := []struct {
		name         string
		resourceName string
		setupFunc    func(*HookFactory)
		expectExists bool
	}{
		{
			name:         "get existing hooks",
			resourceName: "users",
			setupFunc: func(f *HookFactory) {
				f.Register("users", &NoOpHooks[testModel]{})
			},
			expectExists: true,
		},
		{
			name:         "get non-existent hooks",
			resourceName: "missing",
			setupFunc:    func(f *HookFactory) {},
			expectExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewHookFactory()
			tt.setupFunc(factory)

			hooks, exists := factory.GetHooks(tt.resourceName)
			if exists != tt.expectExists {
				t.Errorf("Expected exists=%v, got %v", tt.expectExists, exists)
			}
			if !exists && hooks != nil {
				t.Error("Expected nil hooks when not exists")
			}
		})
	}
}

func TestGetHooksTyped(t *testing.T) {
	tests := []struct {
		name            string
		resourceName    string
		setupFunc       func(*HookFactory)
		expectError     bool
		expectNoOpHooks bool
	}{
		{
			name:         "get typed hooks successfully",
			resourceName: "users",
			setupFunc: func(f *HookFactory) {
				f.Register("users", &NoOpHooks[testModel]{})
			},
			expectError:     false,
			expectNoOpHooks: false,
		},
		{
			name:            "get typed hooks for non-existent resource returns NoOpHooks",
			resourceName:    "missing",
			setupFunc:       func(f *HookFactory) {},
			expectError:     false,
			expectNoOpHooks: true,
		},
		{
			name:         "get typed hooks with wrong type returns error",
			resourceName: "users",
			setupFunc: func(f *HookFactory) {
				f.Register("users", &NoOpHooks[string]{}) // Register string, but we'll request testModel
			},
			expectError:     true,
			expectNoOpHooks: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewHookFactory()
			tt.setupFunc(factory)

			hooks, err := GetHooksTyped[testModel](factory, tt.resourceName)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for type mismatch")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if hooks == nil {
					t.Fatal("Expected non-nil hooks")
				}

				if tt.expectNoOpHooks {
					if _, ok := hooks.(NoOpHooks[testModel]); !ok {
						t.Error("Expected NoOpHooks to be returned for non-existent resource")
					}
				}
			}
		})
	}
}

func TestHookFactory_ListRegistered(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(*HookFactory)
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "empty factory",
			setupFunc:     func(f *HookFactory) {},
			expectedCount: 0,
			expectedNames: []string{},
		},
		{
			name: "single resource",
			setupFunc: func(f *HookFactory) {
				f.Register("users", &NoOpHooks[testModel]{})
			},
			expectedCount: 1,
			expectedNames: []string{"users"},
		},
		{
			name: "multiple resources",
			setupFunc: func(f *HookFactory) {
				f.Register("users", &NoOpHooks[testModel]{})
				f.Register("posts", &NoOpHooks[testModel]{})
				f.Register("comments", &NoOpHooks[testModel]{})
			},
			expectedCount: 3,
			expectedNames: []string{"users", "posts", "comments"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewHookFactory()
			tt.setupFunc(factory)

			resources := factory.ListRegistered()

			if len(resources) != tt.expectedCount {
				t.Errorf("Expected %d resources, got %d", tt.expectedCount, len(resources))
			}

			// Check all expected names are present (order doesn't matter for map)
			resourceMap := make(map[string]bool)
			for _, name := range resources {
				resourceMap[name] = true
			}

			for _, expectedName := range tt.expectedNames {
				if !resourceMap[expectedName] {
					t.Errorf("Expected resource %s not found in list", expectedName)
				}
			}
		})
	}
}

func TestHookFactory_HasHooks(t *testing.T) {
	tests := []struct {
		name         string
		resourceName string
		setupFunc    func(*HookFactory)
		expected     bool
	}{
		{
			name:         "has hooks for existing resource",
			resourceName: "users",
			setupFunc: func(f *HookFactory) {
				f.Register("users", &NoOpHooks[testModel]{})
			},
			expected: true,
		},
		{
			name:         "no hooks for non-existent resource",
			resourceName: "missing",
			setupFunc:    func(f *HookFactory) {},
			expected:     false,
		},
		{
			name:         "no hooks after removal",
			resourceName: "users",
			setupFunc: func(f *HookFactory) {
				f.Register("users", &NoOpHooks[testModel]{})
				f.Remove("users")
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewHookFactory()
			tt.setupFunc(factory)

			result := factory.HasHooks(tt.resourceName)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHookFactory_Remove(t *testing.T) {
	factory := NewHookFactory()

	// Register multiple resources
	factory.Register("users", &NoOpHooks[testModel]{})
	factory.Register("posts", &NoOpHooks[testModel]{})
	factory.Register("comments", &NoOpHooks[testModel]{})

	// Remove one
	factory.Remove("posts")

	// Verify removed
	if factory.HasHooks("posts") {
		t.Error("Expected posts to be removed")
	}

	// Verify others still exist
	if !factory.HasHooks("users") {
		t.Error("Expected users to still exist")
	}
	if !factory.HasHooks("comments") {
		t.Error("Expected comments to still exist")
	}
}

func TestHookFactory_RemoveNonExistent(t *testing.T) {
	factory := NewHookFactory()

	// Remove non-existent resource should not panic
	factory.Remove("missing")

	// Verify factory is still functional
	factory.Register("users", &NoOpHooks[testModel]{})
	if !factory.HasHooks("users") {
		t.Error("Factory should still be functional after removing non-existent resource")
	}
}

func TestHookFactory_Clear(t *testing.T) {
	factory := NewHookFactory()

	// Register multiple resources
	factory.Register("users", &NoOpHooks[testModel]{})
	factory.Register("posts", &NoOpHooks[testModel]{})
	factory.Register("comments", &NoOpHooks[testModel]{})

	// Clear all
	factory.Clear()

	// Verify all removed
	if len(factory.ListRegistered()) != 0 {
		t.Errorf("Expected empty registry, got %d items", len(factory.ListRegistered()))
	}

	if factory.HasHooks("users") || factory.HasHooks("posts") || factory.HasHooks("comments") {
		t.Error("Expected all resources to be cleared")
	}
}

func TestHookFactory_ClearAndReuse(t *testing.T) {
	factory := NewHookFactory()

	// Register, clear, and register again
	factory.Register("users", &NoOpHooks[testModel]{})
	factory.Clear()
	factory.Register("posts", &NoOpHooks[testModel]{})

	// Verify factory is functional after clear
	if !factory.HasHooks("posts") {
		t.Error("Factory should be functional after clear")
	}
	if factory.HasHooks("users") {
		t.Error("Old resources should not exist after clear")
	}
}

func TestGlobalFactory(t *testing.T) {
	// Get global factory
	factory1 := GlobalFactory()
	if factory1 == nil {
		t.Fatal("Expected non-nil global factory")
	}

	// Get again, should be same instance
	factory2 := GlobalFactory()
	if factory1 != factory2 {
		t.Error("Expected same global factory instance")
	}
}

func TestRegisterGlobal(t *testing.T) {
	// Clear global factory first
	GlobalFactory().Clear()

	// Register to global factory
	RegisterGlobal("users", &NoOpHooks[testModel]{})

	// Verify registered
	if !GlobalFactory().HasHooks("users") {
		t.Error("Expected users to be registered in global factory")
	}

	// Clean up
	GlobalFactory().Clear()
}

func TestGetGlobal(t *testing.T) {
	// Clear global factory first
	GlobalFactory().Clear()

	// Register to global factory
	hooks := &NoOpHooks[testModel]{}
	RegisterGlobal("users", hooks)

	// Get from global factory
	retrievedHooks, exists := GetGlobal("users")
	if !exists {
		t.Error("Expected to find hooks in global factory")
	}
	if retrievedHooks == nil {
		t.Error("Expected non-nil hooks")
	}

	// Get non-existent
	_, exists = GetGlobal("missing")
	if exists {
		t.Error("Expected missing hooks to not exist")
	}

	// Clean up
	GlobalFactory().Clear()
}

func TestGlobalFactory_Isolation(t *testing.T) {
	// Clear global factory
	GlobalFactory().Clear()

	// Register to global
	RegisterGlobal("global-resource", &NoOpHooks[testModel]{})

	// Create local factory
	localFactory := NewHookFactory()
	localFactory.Register("local-resource", &NoOpHooks[testModel]{})

	// Verify isolation
	if localFactory.HasHooks("global-resource") {
		t.Error("Local factory should not have global resources")
	}
	if !GlobalFactory().HasHooks("global-resource") {
		t.Error("Global factory should have global resources")
	}
	if GlobalFactory().HasHooks("local-resource") {
		t.Error("Global factory should not have local resources")
	}

	// Clean up
	GlobalFactory().Clear()
}

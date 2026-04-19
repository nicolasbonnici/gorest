package hooks

import (
	"fmt"
	"sync"

	"github.com/nicolasbonnici/gorest/rbac"
)

type HookFactory struct {
	registry map[string]interface{}
}

func NewHookFactory() *HookFactory {
	return &HookFactory{
		registry: make(map[string]interface{}),
	}
}
func (f *HookFactory) Register(resourceName string, hooks interface{}) {
	f.registry[resourceName] = hooks
}
func (f *HookFactory) GetHooks(resourceName string) (interface{}, bool) {
	hooks, exists := f.registry[resourceName]
	return hooks, exists
}
func GetHooksTyped[T any](f *HookFactory, resourceName string) (Hooks[T], error) {
	hooksInterface, exists := f.GetHooks(resourceName)
	if !exists {
		return NewNoOpHooks[T](), nil
	}
	hooks, ok := hooksInterface.(Hooks[T])
	if !ok {
		return nil, fmt.Errorf("hooks for resource %s are not of the expected type", resourceName)
	}
	return hooks, nil
}
func (f *HookFactory) ListRegistered() []string {
	resources := make([]string, 0, len(f.registry))
	for name := range f.registry {
		resources = append(resources, name)
	}
	return resources
}
func (f *HookFactory) Clear() {
	f.registry = make(map[string]interface{})
}
func (f *HookFactory) Remove(resourceName string) {
	delete(f.registry, resourceName)
}
func (f *HookFactory) HasHooks(resourceName string) bool {
	_, exists := f.registry[resourceName]
	return exists
}

var globalFactory *HookFactory

func GlobalFactory() *HookFactory {
	if globalFactory == nil {
		globalFactory = NewHookFactory()
	}
	return globalFactory
}
func RegisterGlobal(resourceName string, hooks interface{}) {
	GlobalFactory().Register(resourceName, hooks)
}
func GetGlobal(resourceName string) (interface{}, bool) {
	return GlobalFactory().GetHooks(resourceName)
}

// Global RBAC configuration
var (
	globalRBACConfig   rbac.Config
	globalRBACConfigMu sync.RWMutex
	rbacConfigSet      bool
)

// SetGlobalRBACConfig sets the global RBAC configuration for all hooks
func SetGlobalRBACConfig(config rbac.Config) {
	globalRBACConfigMu.Lock()
	defer globalRBACConfigMu.Unlock()
	globalRBACConfig = config
	rbacConfigSet = true
}

// GetGlobalRBACConfig returns the global RBAC configuration
// If not set, returns default configuration
func GetGlobalRBACConfig() rbac.Config {
	globalRBACConfigMu.RLock()
	defer globalRBACConfigMu.RUnlock()
	if !rbacConfigSet {
		return rbac.DefaultConfig()
	}
	return globalRBACConfig
}

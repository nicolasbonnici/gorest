package plugin

// PluginRegistry stores all loaded plugins.
// Plugins are NOT automatically applied; user must explicitly call plugin.Handler() and register it.
type PluginRegistry struct {
	plugins map[string]Plugin
}

func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]Plugin),
	}
}

// Register adds a plugin to the registry.
// Does NOT apply the plugin to any routes - that must be done manually.
func (r *PluginRegistry) Register(plugin Plugin) {
	r.plugins[plugin.Name()] = plugin
}

// Get retrieves a plugin by name.
func (r *PluginRegistry) Get(name string) (Plugin, bool) {
	plugin, exists := r.plugins[name]
	return plugin, exists
}

// GetAll returns all registered plugins.
func (r *PluginRegistry) GetAll() map[string]Plugin {
	return r.plugins
}

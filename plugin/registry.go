package plugin

// Plugins are NOT automatically applied; user must explicitly call plugin.Handler() and register it.
type PluginRegistry struct {
	plugins map[string]Plugin
}

func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]Plugin),
	}
}

func (r *PluginRegistry) Register(plugin Plugin) {
	r.plugins[plugin.Name()] = plugin
}

func (r *PluginRegistry) Get(name string) (Plugin, bool) {
	plugin, exists := r.plugins[name]
	return plugin, exists
}

func (r *PluginRegistry) GetAll() map[string]Plugin {
	return r.plugins
}

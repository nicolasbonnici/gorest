package builtin

import (
	"github.com/nicolasbonnici/gorest/plugin"
)

// init registers all built-in plugins with the plugin system
func init() {
	// Register global plugins
	plugin.RegisterGlobalPluginFactory("requestid", func() plugin.GlobalPlugin {
		return NewRequestIDPlugin()
	})
	plugin.RegisterGlobalPluginFactory("security", func() plugin.GlobalPlugin {
		return NewSecurityPlugin()
	})
	plugin.RegisterGlobalPluginFactory("contenttype", func() plugin.GlobalPlugin {
		return NewContentTypePlugin()
	})
	plugin.RegisterGlobalPluginFactory("logger", func() plugin.GlobalPlugin {
		return NewLoggerPlugin()
	})
	plugin.RegisterGlobalPluginFactory("cors", func() plugin.GlobalPlugin {
		return NewCORSPlugin()
	})
	plugin.RegisterGlobalPluginFactory("ratelimit", func() plugin.GlobalPlugin {
		return NewRateLimitPlugin()
	})

	// Register route plugins
	plugin.RegisterRoutePluginFactory("auth", func() plugin.RoutePlugin {
		return NewAuthPlugin()
	})
}

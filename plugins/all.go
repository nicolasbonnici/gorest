package plugins

import (
	_ "github.com/nicolasbonnici/gorest/plugins/auth"
	_ "github.com/nicolasbonnici/gorest/plugins/contenttype"
	_ "github.com/nicolasbonnici/gorest/plugins/cors"
	_ "github.com/nicolasbonnici/gorest/plugins/logger"
	_ "github.com/nicolasbonnici/gorest/plugins/ratelimit"
	_ "github.com/nicolasbonnici/gorest/plugins/requestid"
	_ "github.com/nicolasbonnici/gorest/plugins/security"
)

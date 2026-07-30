package processor

import (
	"sync/atomic"

	"github.com/nicolasbonnici/gorest/crud"
)

var defaultCountMode atomic.Value

// SetDefaultCountMode records the application-wide count strategy read from
// pagination.count. GoREST calls it once during startup, before any request is
// served, so every processor built afterwards — generated resources and plugins
// alike — adopts it without threading the value through their constructors.
//
// Being process-wide, it cannot differ between two GoREST instances sharing a
// process. Pass ProcessorConfig.CountMode explicitly to override it per
// processor.
func SetDefaultCountMode(mode crud.CountMode) {
	if mode == "" {
		mode = crud.CountExact
	}
	defaultCountMode.Store(mode)
}

// DefaultCountMode returns the application-wide count strategy, or
// crud.CountExact when startup has not set one.
func DefaultCountMode() crud.CountMode {
	if mode, ok := defaultCountMode.Load().(crud.CountMode); ok {
		return mode
	}
	return crud.CountExact
}

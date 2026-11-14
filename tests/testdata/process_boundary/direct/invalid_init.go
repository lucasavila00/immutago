package direct

import "example.com/consumer/api"

// callInit attempts to call an imported process's runtime entry point.
func callInit(pid api.Handle) { pid.Proc().Init() }

// callCustomInit attempts to call an imported explicit initializer.
func callCustomInit(pid api.ReadyHandle) { pid.Proc().Init() }

package direct

import "example.com/consumer/api"

// callInit attempts to call an imported service's runtime entry point.
func callInit(ref api.Handle) { ref.Borrow().Init() }

// callCustomInit attempts to call an imported explicit initializer.
func callCustomInit(ref api.ReadyHandle) { ref.Borrow().Init() }

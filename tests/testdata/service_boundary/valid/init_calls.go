package valid

import "example.com/consumer/api"

// callInit calls an imported service's runtime entry point.
func callInit(ref api.Handle) { ref.Borrow().Init(nil) }

// callCustomInit calls an imported explicit initializer.
func callCustomInit(ref api.ReadyHandle) { ref.Borrow().Init(nil) }

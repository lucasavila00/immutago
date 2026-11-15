package dependency

import "example.com/consumer/api"

// Hidden removes the dependency's named identity through an imported alias.
type Hidden api.Dependency

// Holder stores a dependency in an unmarked struct.
type Holder struct{ Value api.Dependency }

// Slice stores imported dependency aliases in a native container.
func Slice() { var values []api.Dependency; _ = len(values) }

// Unwrap passes a dependency through an interface outside a service.
func Unwrap(value api.Dependency) *int {
	var stored any = value
	return stored.(*int)
}

// DirectUnwrap returns the pointer through an ordinary direct call.
func DirectUnwrap(value api.Dependency) *int { return value }

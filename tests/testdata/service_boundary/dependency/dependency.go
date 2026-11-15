package dependency

import "example.com/consumer/api"

// Hidden removes the dependency's named identity through an imported alias.
type Hidden api.Dependency

// Holder stores a dependency in an unmarked struct.
type Holder struct{ Value api.Dependency }

// Slice stores imported dependency aliases in a native container.
func Slice() { var values []api.Dependency; _ = len(values) }

// Unwrap calls Value through an interface outside a service.
func Unwrap(value api.Dependency) *int {
	var getter interface{ Value() *int } = value
	return getter.Value()
}

// DirectUnwrap reads the wrapper through an ordinary direct method call.
func DirectUnwrap(value api.Dependency) *int { return value.Value() }

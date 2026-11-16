package invalid

import "example.com/consumer/api"

// CounterHolder has a replaceable handle field.
type CounterHolder struct{ api.Counter }

// Pointer cannot take the implicit address of a replaceable Frozen field.
func Pointer(value CounterHolder) *api.Counter { return value.Pointer() }

// BoundPointer cannot save that implicit address in a method value.
func BoundPointer(value CounterHolder) func() *api.Counter { return value.Pointer }

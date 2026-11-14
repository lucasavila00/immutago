package process

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Anonymous has an anonymous process type.
var Anonymous struct{ proc.Process[proc.None] }

// Generic is a generic process.
type Generic[T any] struct{ proc.Process[T] }

// NamedMarker names the marker field.
type NamedMarker struct{ Marker proc.Process[proc.None] }

// MarkedProcess also embeds an immutable marker.
type MarkedProcess struct {
	proc.Process[proc.None]
	im.Shared
}

// ProtectedField stores a frozen collection.
type ProtectedField struct {
	proc.Process[proc.None]
	limits im.FrozenMap[string, int]
}

// GenericPID takes a PID of a type parameter.
func GenericPID[T any](pid proc.PID[T]) {}

// MapArguments has a native map as its arguments.
type MapArguments struct{ proc.Process[map[string]int] }

// ErrorArguments has an interface as its arguments.
type ErrorArguments struct{ proc.Process[error] }

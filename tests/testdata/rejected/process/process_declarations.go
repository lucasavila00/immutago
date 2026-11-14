package process

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Anonymous has an anonymous process type.
var Anonymous struct{ proc.Process }

// Generic is a generic process.
type Generic[T any] struct{ proc.Process }

// NamedMarker names the marker field.
type NamedMarker struct{ Marker proc.Process }

// MarkedProcess also embeds an immutable marker.
type MarkedProcess struct {
	proc.Process
	im.Shared
}

// ProtectedField stores a frozen collection.
type ProtectedField struct {
	proc.Process
	limits im.FrozenMap[string, int]
}

// GenericPID takes a PID of a type parameter.
func GenericPID[T any](pid proc.PID[T]) {}

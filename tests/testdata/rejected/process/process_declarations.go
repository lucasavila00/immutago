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

// Config is an immutable value that stays local.
type Config struct {
	im.Scoped
	Version int
}

// ScopedArguments has a Scoped pointer as its arguments.
type ScopedArguments struct{ proc.Process[*Config] }

// CallbackArguments has a callback that carries protected storage as its arguments.
type CallbackArguments struct{ proc.Process[func(*Config)] }

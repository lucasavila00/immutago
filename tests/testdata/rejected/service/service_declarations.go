package service

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Anonymous has an anonymous service type.
var Anonymous struct {
	supervise.Service[supervise.NoArgs]
}

// Generic is a generic service.
type Generic[T any] struct{ supervise.Service[T] }

// NamedMarker names the marker field.
type NamedMarker struct {
	Marker supervise.Service[supervise.NoArgs]
}

// MarkedService also embeds an immutable marker.
type MarkedService struct {
	supervise.Service[supervise.NoArgs]
	im.Shared
}

// ProtectedField stores a frozen collection.
type ProtectedField struct {
	supervise.Service[supervise.NoArgs]
	limits im.FrozenMap[string, int]
}

// GenericRef takes a Ref of a type parameter.
func GenericRef[T any](ref supervise.Ref[T]) {}

// MapArguments has a native map as its arguments.
type MapArguments struct {
	supervise.Service[map[string]int]
}

// ErrorArguments has an interface as its arguments.
type ErrorArguments struct{ supervise.Service[error] }

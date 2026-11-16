package valid

import "github.com/lucasavila00/immutago/pkg/im"

type genericState[T any] struct{ value T }

// GenericHandle uses the same opaque shape for each concrete type argument.
type GenericHandle[T any] struct {
	im.Frozen
	state im.UncheckedValue[genericState[T]]
}

// Pointer returns the existing Frozen wrapper pointer.
func (h *GenericHandle[T]) Pointer() *GenericHandle[T] { return h }

// FreezeGenericSlice calls a method on a stored pointer before freezing the builder.
func FreezeGenericSlice() *GenericHandle[int] {
	values := []*GenericHandle[int]{new(GenericHandle[int])}
	kept := values[0].Pointer()
	_ = im.FreezeSlice(values)
	return kept
}

// FreezeGenericMap uses the same stored pointer rule through an alias and map entry.
func FreezeGenericMap() *GenericHandle[int] {
	type Alias = GenericHandle[int]
	values := map[int]*Alias{0: new(Alias)}
	kept := values[0].Pointer()
	_ = im.FreezeMap(values)
	return kept
}

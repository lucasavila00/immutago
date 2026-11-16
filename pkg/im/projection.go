package im

// Projection holds a value for local reads. Its zero value contains the zero T.
// The checker proves T and restricts storage and transport of the projection.
// Copies copy T by value and share its immutable backing storage.
// See the complete forms at
// https://github.com/lucasavila00/immutago/blob/main/docs/reference/projection-primitives.md.
type Projection[T any] struct {
	value T
}

// ProjectionOf returns a local projection of value. It copies T by value without copying
// backing storage or allocating. Run immutago to enforce the local projection rules.
func ProjectionOf[T any](value T) Projection[T] { return Projection[T]{value: value} }

// CopyRecord allocates a private root copy and returns a local pointer projection.
// It copies T by value and shares immutable backing storage. A nonzero-sized T
// needs one allocation. The checker requires a Frozen record T.
func CopyRecord[T any](value Projection[T]) Projection[*T] { return ProjectionOf(new(value.value)) }

// Unchecked returns the selected Frozen value for use outside the local projection.
// It copies T by value without copying backing storage or allocating. A pointer
// result is the same pointer. The caller controls retention and must not change
// its target through reflection, unsafe code, or another API.
// The result keeps its normal type and write rules.
func (p Projection[T]) Unchecked() T { return p.value }

// Value returns the value from a projection. T must be Frozen and a read result.
// See https://github.com/lucasavila00/immutago/blob/main/docs/reference/types.md for the complete type list.
// Value does not allocate.
func Value[T any](value Projection[T]) T { return value.value }

// IsNil reports whether a pointer projection contains nil without exposing the pointer.
func IsNil[T any](value Projection[*T]) bool { return value.value == nil }

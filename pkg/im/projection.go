package im

// Projection holds a value for local reads. Its zero value contains the zero T.
// The checker proves T and restricts storage and transport of the projection.
// Copies copy T by value and share its immutable backing storage.
// See https://github.com/lucasavila00/immutago/blob/main/docs/projections.md for the complete forms.
type Projection[T any] struct {
	value T
}

// ProjectionOf returns a local projection of value. It copies T by value without copying
// backing storage or allocating. Run immutago to enforce the local projection rules.
func ProjectionOf[T any](value T) Projection[T] { return Projection[T]{value: value} }

// CopyProjection allocates a private root copy and returns a local pointer projection.
// It copies T by value and shares immutable backing storage. A nonzero-sized T
// needs one allocation. The checker requires a Frozen record T.
func CopyProjection[T any](value Projection[T]) Projection[*T] { return ProjectionOf(new(value.value)) }

// Read supplies direct field access to generated accessors without copying backing storage.
// Application code uses the generated projection methods. The checker controls Read access.
func (p Projection[T]) Read() T { return p.value }

// Copy returns the value from a projection. The checker permits only Copied types,
// including opaque service references. Copy does not allocate.
func Copy[T any](value Projection[T]) T { return value.value }

// IsNil reports whether a pointer projection contains nil without exposing the pointer.
func IsNil[T any](value Projection[*T]) bool { return value.value == nil }

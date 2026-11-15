package im

// View holds a value for local reads. Its zero value contains the zero T.
// The checker proves T and restricts storage and transport of the view.
// Copies copy T by value and share its immutable backing storage.
// See https://github.com/lucasavila00/immutago/blob/main/docs/views.md for the complete forms.
type View[T any] struct {
	value T
}

// ViewOf returns a local view of value. It copies T by value without copying
// backing storage or allocating. Run immutago to enforce the local view rules.
func ViewOf[T any](value T) View[T] { return View[T]{value: value} }

// CopyView allocates a private root copy and returns a local pointer view.
// It copies T by value and shares immutable backing storage. A nonzero-sized T
// needs one allocation. The checker requires a Frozen record T.
func CopyView[T any](value View[T]) View[*T] { return ViewOf(new(value.value)) }

// Select calls project once, before returning, and wraps its result in a View.
// The checker permits only the listed field and array-index paths in project.
// Select does not keep project or copy the selected value's backing storage.
func Select[T, F any](value View[T], project func(T) F) View[F] {
	return ViewOf(project(value.value))
}

// Copy returns the value from a view. The checker permits only Copied types,
// including opaque service references. Copy does not allocate.
func Copy[T any](value View[T]) T { return value.value }

// IsNil reports whether a pointer view contains nil without exposing the pointer.
func IsNil[T any](value View[*T]) bool { return value.value == nil }

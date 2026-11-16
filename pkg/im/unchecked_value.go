package im

// UncheckedValue holds an opaque pointer whose access rules belong to its declaring package.
// The target can be mutable. The checker does not prove its state or synchronization.
// UncheckedValue is Frozen and a read result, but not Copied.
// Any package can store, copy, pass, and return the handle.
// Only the package that declares T can use Trust and Unwrap.
// See https://github.com/lucasavila00/immutago/blob/main/docs/reference/unchecked-values.md for the complete forms.
type UncheckedValue[T any] struct {
	value *T
}

// Trust stores value without copying its target or adding synchronization.
// Only the package that declares T can use Trust.
func Trust[T any](value *T) UncheckedValue[T] { return UncheckedValue[T]{value: value} }

// Unwrap returns the wrapped pointer, or nil for the zero UncheckedValue.
// Only the package that declares T can use Unwrap. Its API controls access to T.
func Unwrap[T any](value UncheckedValue[T]) *T { return value.value }

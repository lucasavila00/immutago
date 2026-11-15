package im

// UncheckedValue holds a pointer to data whose immutability the analyzer does not prove.
// The package that declares T promises never to change the data after Trust.
//
// UncheckedValue is an allowed stored type. Any package can store, copy, and pass it.
// Only the package that declares T can use Trust and Unwrap, so other packages
// reach the value only through the API of that package.
// See https://github.com/lucasavila00/immutago/blob/main/docs/unchecked-values.md for the complete forms.
type UncheckedValue[T any] struct {
	value *T
}

// Trust wraps value with an unchecked promise that it is immutable.
// It does not copy or freeze the data. Only the package that declares T can
// use Trust, and that package must not change the value after this call.
func Trust[T any](value *T) UncheckedValue[T] { return UncheckedValue[T]{value: value} }

// Unwrap returns the wrapped pointer, or nil for the zero UncheckedValue.
// Only the package that declares T can use Unwrap.
func Unwrap[T any](value UncheckedValue[T]) *T { return value.value }

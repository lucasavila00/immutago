package im

// Unchecked holds a pointer to a value that the analyzer does not check.
// It is the one trust boundary of immutago: the analyzer does not prove that
// the value is immutable. The package that declares T trusts its own code to
// never change the value after Trust.
//
// Unchecked is an allowed stored type. Any package can store, copy, and pass
// it. Only the package that declares T can call Trust and Value, so other
// packages reach the value only through the API of that package.
// See https://github.com/lucasavila00/immutago/blob/main/docs/unchecked.md for the complete forms.
type Unchecked[T any] struct {
	value *T
}

// Trust wraps value. The package that declares T must not change the value
// after this call.
func Trust[T any](value *T) Unchecked[T] { return Unchecked[T]{value: value} }

// Value returns the wrapped pointer, or nil for the zero Unchecked.
func (u Unchecked[T]) Value() *T { return u.value }

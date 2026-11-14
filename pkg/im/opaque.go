package im

// Opaque holds a pointer to a value that the analyzer does not look inside.
// It is one of the two trust markers of immutago, with proc.UncheckedDependency: the
// analyzer does not prove that the value is immutable. The package that
// declares T trusts its own code to never change the value after Seal.
//
// Opaque is an allowed stored type. Any package can store, copy, and pass it.
// Only the package that declares T can use Seal and Unseal, so other packages
// reach the value only through the API of that package.
// See https://github.com/lucasavila00/immutago/blob/main/docs/opaque.md for the complete forms.
type Opaque[T any] struct {
	value *T
}

// Seal wraps value. The package that declares T must not change the value
// after this call.
func Seal[T any](value *T) Opaque[T] { return Opaque[T]{value: value} }

// Unseal returns the wrapped pointer, or nil for the zero Opaque.
// Only the package that declares T can use Unseal.
func Unseal[T any](value Opaque[T]) *T { return value.value }

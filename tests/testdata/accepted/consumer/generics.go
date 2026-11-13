package consumer

import "github.com/lucasavila00/immutago/pkg/im"

type Scalar interface{ ~int | ~string }
type EmbeddedScalar interface{ Scalar }
type PositiveScalar interface {
	Scalar
	comparable
}
type Generic[T EmbeddedScalar] struct {
	im.Immut
	Value T
}
type GenericAlias[T Scalar] = im.FrozenSlice[T]
type NamedInts struct {
	im.Immut
	Values [2]int
}

func Build[T PositiveScalar](value T) im.FrozenSlice[T] {
	frozen := im.FreezeSlice([]T{value})
	return frozen
}
func GenericUse() {
	_ = Build(1)
	_ = Build("text")
	_ = Generic[int]{Value: 1}
	var zero GenericAlias[string]
	_ = zero.Len()
	_ = im.FreezeSlice([]NamedInts{{Values: [2]int{1, 2}}})
}

func boxScalar[T ~int | ~string](value T) *T { return &value }

func ScalarPointerCopy() {
	value := Generic[int]{Value: 1}
	pointer := boxScalar(value.Value)
	*pointer = 2
	_ = value.Value
	_ = boxScalar("text")
}

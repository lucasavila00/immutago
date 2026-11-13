package consumer

import "github.com/lucasavila00/immutago/pkg/im"

type Scalar interface{ ~int | ~string }
type EmbeddedScalar interface{ Scalar }
type PositiveScalar interface {
	Scalar
	comparable
}
type Generic[T EmbeddedScalar] struct {
	im.Marker
	Value T
}
type GenericAlias[T Scalar] = im.Slice[T]
type NamedInts struct {
	im.Marker
	Values [2]int
}

func Build[T PositiveScalar](value T) im.Slice[T] { return im.NewSlice(value) }
func GenericUse() {
	_ = Build(1)
	_ = Build("text")
	_ = Generic[int]{Value: 1}
	var zero GenericAlias[string]
	_ = zero.Append("value")
	_ = im.NewSlice(NamedInts{Values: [2]int{1, 2}})
}

func boxScalar[T ~int | ~string](value T) *T { return &value }

func ScalarPointerCopy() {
	value := Generic[int]{Value: 1}
	pointer := boxScalar(value.Value)
	*pointer = 2
	_ = value.Value
	_ = boxScalar("text")
}

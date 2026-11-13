package consumer

import im "github.com/lucasavila00/immutago/pkg/immutable"

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
type NamedInts [2]int

func Build[T PositiveScalar](value T) im.Slice[T] { return im.NewSlice(value) }
func GenericUse() {
	_ = Build(1)
	_ = Build("text")
	_ = Generic[int]{Value: 1}
	var zero GenericAlias[string]
	_ = zero.Append("value")
	_ = im.NewSlice(NamedInts{1, 2})
}

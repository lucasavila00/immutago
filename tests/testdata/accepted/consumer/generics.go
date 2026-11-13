package consumer

import "github.com/lucasavila00/immutago/pkg/im"

type NamedInts struct {
	im.Shared
	Values [2]int
}
type Plain[T any] struct{ Value T }

func boxScalar[T ~int | ~string](value T) *T { return &value }
func ScalarPointerCopy() {
	value := Plain[int]{Value: 1}
	pointer := boxScalar(value.Value)
	*pointer = 2
	_ = value.Value
	_ = boxScalar("text")
	_ = im.FreezeSlice([]NamedInts{{Values: [2]int{1, 2}}})
}

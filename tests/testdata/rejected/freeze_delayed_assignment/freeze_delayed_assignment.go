package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() {
	values := []int{1}
	var frozen im.FrozenSlice[int]
	values[0], frozen = 2, im.FreezeSlice(values)
	_ = frozen.At(0)
}

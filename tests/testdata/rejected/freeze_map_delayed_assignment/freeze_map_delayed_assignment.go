package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() {
	values := map[int]int{1: 1}
	var frozen im.FrozenMap[int, int]
	values[1], frozen = 2, im.FreezeMap(values)
	_, _ = frozen.Get(1)
}

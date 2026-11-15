package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() {
	values := map[int]int{1: 1}
	frozen := im.FreezeMap(values)
	values[1] = 2
	_, _ = frozen.Get(1)
}

package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() {
	values := map[int]int{1: 1}
	alias := values
	_ = alias[1]
	_ = im.FreezeMap(values)
}

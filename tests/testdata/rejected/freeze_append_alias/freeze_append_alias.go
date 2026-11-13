package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() {
	values := make([]int, 1, 2)
	alias := append(values, 2)
	_ = im.FreezeSlice(values)
	_ = alias
}

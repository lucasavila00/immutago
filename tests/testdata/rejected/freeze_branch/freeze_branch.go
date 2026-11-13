package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad(condition bool) {
	values := []int{1}
	if condition {
		_ = im.FreezeSlice(values)
	}
	values[0] = 2
}

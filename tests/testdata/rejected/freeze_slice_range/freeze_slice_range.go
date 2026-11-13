package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() {
	values := []int{1}
	_ = im.FreezeSlice(values)
	for _, v := range values {
		_ = v
	}
}

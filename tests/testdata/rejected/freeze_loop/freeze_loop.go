package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad(count int) {
	values := []int{1}
	for i := 0; i < count; i++ {
		_ = im.FreezeSlice(values)
	}
}

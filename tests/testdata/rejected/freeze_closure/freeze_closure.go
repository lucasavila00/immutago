package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() {
	values := []int{1}
	later := func() { values[0] = 2 }
	_ = im.FreezeSlice(values)
	later()
}

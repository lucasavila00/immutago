package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() {
	values := [][2]int{{1, 2}}
	alias := values[0][:]
	_ = im.FreezeSlice(values)
	alias[0] = 2
}

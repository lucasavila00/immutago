package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() {
	frozen := im.FreezeSlice([]int{1})
	values := frozen.Clone()
	alias := values
	_ = im.FreezeSlice(values)
	_ = alias
}

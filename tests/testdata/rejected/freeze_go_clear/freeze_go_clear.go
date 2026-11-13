package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() { values := []int{1}; go clear(values); _ = im.FreezeSlice(values) }

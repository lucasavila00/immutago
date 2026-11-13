package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() { values := []int{1}; alias := values; _ = alias[0]; _ = im.FreezeSlice(values) }

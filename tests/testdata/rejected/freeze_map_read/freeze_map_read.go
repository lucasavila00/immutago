package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() { values := map[int]int{1: 1}; _ = im.FreezeMap(values); _ = values[1] }

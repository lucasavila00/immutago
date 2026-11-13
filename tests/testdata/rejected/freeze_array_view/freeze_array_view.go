package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() { array := [2]int{1, 2}; values := array[:]; _ = im.FreezeSlice(values) }

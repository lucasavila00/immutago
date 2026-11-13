package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() { values := []int{1}; frozen := im.FreezeSlice(values); values[0] = 2; _ = frozen.At(0) }

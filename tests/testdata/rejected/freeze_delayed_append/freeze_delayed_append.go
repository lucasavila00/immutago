package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() { values := make([]int, 1, 2); values = append(values, im.FreezeSlice(values).At(0)) }

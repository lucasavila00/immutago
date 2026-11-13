package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() { values := []int{1}; _ = im.FreezeSlice(values); values = append(values, 2) }

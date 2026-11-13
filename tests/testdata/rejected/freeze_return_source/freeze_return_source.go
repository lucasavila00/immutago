package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() []int { values := []int{1}; _ = im.FreezeSlice(values); return values }

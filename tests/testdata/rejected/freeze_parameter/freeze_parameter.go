package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad(values []int) { _ = im.FreezeSlice(values) }

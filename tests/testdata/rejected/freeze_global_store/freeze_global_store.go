package rejected

import "github.com/lucasavila00/immutago/pkg/im"

var saved []int

func bad() { values := []int{1}; saved = values; _ = im.FreezeSlice(values) }

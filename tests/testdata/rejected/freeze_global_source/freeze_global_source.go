package rejected

import "github.com/lucasavila00/immutago/pkg/im"

var values = []int{1}
var frozen = im.FreezeSlice(values)

func bad() { values[0] = 2; _ = frozen.At(0) }

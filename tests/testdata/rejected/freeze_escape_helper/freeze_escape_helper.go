package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func touch(values []int) { values[0] = 3 }
func bad()               { values := []int{1}; touch(values); _ = im.FreezeSlice(values) }

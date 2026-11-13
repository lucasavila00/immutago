package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() { values := []int{1}; p := &values[0]; _ = im.FreezeSlice(values); *p = 2 }

package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() { values := map[string]int{"x": 1}; _ = im.FreezeMap(values); delete(values, "x") }

package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() {
	values := map[string]int{"x": 1}
	alias := values
	_ = alias["x"]
	_ = im.FreezeMap(values)
}

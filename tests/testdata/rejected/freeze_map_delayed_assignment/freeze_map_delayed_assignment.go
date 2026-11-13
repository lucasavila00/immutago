package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() {
	values := map[string]int{"x": 1}
	var frozen im.FrozenMap[string, int]
	values["x"], frozen = 2, im.FreezeMap(values)
	_, _ = frozen.Get("x")
}

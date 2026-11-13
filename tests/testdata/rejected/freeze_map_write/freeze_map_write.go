package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func bad() {
	values := map[string]int{"x": 1}
	frozen := im.FreezeMap(values)
	values["x"] = 2
	_, _ = frozen.Get("x")
}

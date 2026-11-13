package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Child struct {
	im.Marker
	X int
}
type Good struct {
	im.Marker
	X, Y  int
	Child Child
	Array [2]int
}

func read(p *Good) int { return p.X }

func build() {
	frozen := im.FreezeMap(map[string]Good{})
	values := frozen.Clone()
	values["a"] = Good{X: 1}
	_ = im.FreezeMap(values)
}

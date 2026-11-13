package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Child struct {
	im.Immut
	X int
}
type Good struct {
	im.Immut
	X, Y  int
	Child Child
	Array [2]int
}

func read(p *Good) int { return p.X }

func bad(p *Good) *Good { return p }

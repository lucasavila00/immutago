package accepted

import "github.com/lucasavila00/immutago/pkg/im"

type Child struct {
	im.Shared
	X int
}
type Good struct {
	im.Shared
	X, Y  int
	Child Child
	Array [2]int
}

func read(p *Good) int { return p.X }

func forward(p *Good) *Good { return p }

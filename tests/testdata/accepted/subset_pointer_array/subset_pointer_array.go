package accepted

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

func readArray(values [1]*Good) { _ = values }

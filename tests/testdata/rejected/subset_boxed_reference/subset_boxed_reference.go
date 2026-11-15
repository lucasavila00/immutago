//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package rejected

import "github.com/lucasavila00/immutago/pkg/im"

type Child struct {
	im.Frozen
	X int
}
type Good struct {
	im.Frozen
	X, Y  int
	Child Child
	Array [2]int
}

func read(p *Good) int { return p.X }

func bad(v Good) { var boxed any = new(v); _ = boxed }

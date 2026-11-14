package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

type ProjectionValue struct {
	im.Shared
	Values [1][1]int
}

func projectionReads(p *ProjectionValue, ordinary [1][1][1]int) int {
	builder := []int{1}
	x := (p).Values[0][0]
	y := (*p).Values[0][0]
	z := (builder)[0]
	plain := (ordinary)[0][0][0]
	_ = im.FreezeSlice(builder)
	return x + y + z + plain
}

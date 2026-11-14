package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

type ProjectionValue struct {
	im.Shared
	Values [1][1]int
}

func projectionWrites(p *ProjectionValue) {
	(*p).Values[0][0] = 2
	builder := []int{1}
	a, b := (builder)[0], 2
	_, _ = a, b
	_ = im.FreezeSlice(builder)
}

package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

type ProjectionValue struct {
	im.Shared
	Values [1][1]int
}

type PlainProjection struct {
	Inner struct{ Value int }
}

func projectionReads(p *ProjectionValue, ordinary [1][1][1]int, plainFields PlainProjection) int {
	builder := []int{1}
	x := (p).Values[0][0]
	y := (*p).Values[0][0]
	z := (builder)[0]
	plain := (ordinary)[0][0][0]
	field := (plainFields).Inner.Value
	_ = im.FreezeSlice(builder)
	return x + y + z + plain + field
}

package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

type PointerValue struct {
	im.Frozen
	N int
}

func returnArrayPointer() (result *[1]PointerValue) {
	return
}

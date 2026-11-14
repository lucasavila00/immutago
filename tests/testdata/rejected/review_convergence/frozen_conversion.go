package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

type UnmarkedFrozen im.FrozenSlice[int]

func convertFrozen(value im.FrozenSlice[int]) {
	converted := UnmarkedFrozen(value)
	_ = converted
}

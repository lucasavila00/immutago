package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

func rangeExistingTargets() {
	values := []int{1}
	index := 0
	value := 0
	for index, value = range values {
	}
	_ = index
	_ = value
	_ = im.FreezeSlice(values)
}

package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

func builderWithScalar() {
	values, count := []int{1}, 2
	_ = count
	_ = im.FreezeSlice(values)
}

func twoBuilders() {
	left, right := []int{1}, []int{2}
	_ = im.FreezeSlice(left)
	_ = im.FreezeSlice(right)
}

func makeBuilderWithScalar() {
	values, count := make([]int, 1), 2
	_ = count
	_ = im.FreezeSlice(values)
}

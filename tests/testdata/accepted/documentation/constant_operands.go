package main

import "github.com/lucasavila00/immutago/pkg/im"

func constantBuilderOperands() {
	values := []int{1}
	const before = len([1][]int{values})
	_ = im.FreezeSlice(values)
	const after = len([1][]int{values})
	const address = len([1]*[]int{&values})
	const index = len([1]int{values[100]})
	println(before, after, address, index)
}

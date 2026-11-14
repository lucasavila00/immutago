package main

import "github.com/lucasavila00/immutago/pkg/im"

func constantBuilderOperands() {
	values := []int{1}
	before := len(values)
	_ = im.FreezeSlice(values)
	const known = 1
	println(before, known)
}

func constantClosureOperands() {
	values := [1]ArrayLengthValue{}
	count := len(values)
	read := func() int {
		return count
	}
	println(read())
}

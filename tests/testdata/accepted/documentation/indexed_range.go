package main

import "github.com/lucasavila00/immutago/pkg/im"

type IndexedValue struct {
	im.Shared
	Index int
}

func indexedRangeTarget() {
	values := []IndexedValue{{Index: 0}}
	positions := []int{0}
	for positions[values[0].Index] = range positions {
	}
	_ = im.FreezeSlice(values)
}

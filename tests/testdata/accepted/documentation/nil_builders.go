package main

import "github.com/lucasavila00/immutago/pkg/im"

type nilInts []int

type nilShared struct {
	im.Shared
	N int
}

// nilBuilders uses zero declarations or bare nil without a conversion.
func nilBuilders() {
	var values []int
	values = append(values, 1)
	frozen := im.FreezeSlice(values)
	println(frozen.At(0))
	var scores map[string]int = nil
	_ = im.FreezeMap(scores)
	var named nilInts
	_ = im.FreezeSlice(named)
	var objects []nilShared = nil
	objects = append(objects, nilShared{N: 2})
	_ = im.FreezeSlice(objects)
	_ = im.FreezeSlice[int](nil)
	_ = im.FreezeMap[string, int](nil)
	_ = im.FreezeSlice[nilShared]((nil))
}

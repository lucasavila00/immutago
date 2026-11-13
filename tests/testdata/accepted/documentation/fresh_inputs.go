package main

import "github.com/lucasavila00/immutago/pkg/im"

// freshInputs checks the direct fresh Freeze forms in the language rules.
func freshInputs() {
	emptySlice := im.FreezeSlice[int](nil)
	emptyMap := im.FreezeMap[string, int](nil)
	copiedSlice := im.FreezeSlice(emptySlice.Clone())
	copiedMap := im.FreezeMap(emptyMap.Clone())
	allocatedSlice := im.FreezeSlice(make([]int, 2))
	allocatedMap := im.FreezeMap(make(map[string]int))
	literalSlice := im.FreezeSlice([]int{1, 2})
	literalMap := im.FreezeMap(map[string]int{"alice": 10})
	println(copiedSlice.Len(), copiedMap.Len(), allocatedSlice.Len(), allocatedMap.Len())
	println(literalSlice.Len(), literalMap.Len())
}

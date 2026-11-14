package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

type PointerValue struct {
	im.Shared
	N int
}

var frozenPointer *im.FrozenSlice[int]

func acceptFrozenPointer(*im.FrozenSlice[int]) {}
func passNilPointer()                          { acceptFrozenPointer(nil) }
func returnArrayPointer() (result *[1]PointerValue) {
	return
}

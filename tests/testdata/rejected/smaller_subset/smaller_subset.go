package smaller_subset

import "github.com/lucasavila00/immutago/pkg/im"

type Node struct {
	im.Shared
	Value int
	Next  *Node
}
type Local struct {
	im.LocalRef
	Value int
}
type Box[T ~int | ~string] struct {
	im.Shared
	Value T
	Next  *Box[T]
}
type GenericAlias[T ~int] = im.FrozenSlice[T]

func genericBuild[T ~int](value T) im.FrozenSlice[T] {
	frozen := im.FreezeSlice([]T{value})
	return frozen
}
func arrayReference()      { p := &[2]Node{}; _ = (*p)[0].Value }
func collectionReference() { p := new(im.FrozenSlice[int]); _ = p.Len() }
func capturedReference() func() int {
	shared := &Node{Value: 1}
	return func() int { return shared.Value }
}
func capturedValue() func() int {
	value := Node{Value: 1}
	return func() int { return value.Value }
}
func capturedLocalValue() func() int {
	value := Local{Value: 1}
	return func() int { return value.Value }
}
func readNode(node *Node) int { return node.Value }
func scheduledReads() {
	shared := &Node{Value: 1}
	go readNode(shared)
	defer readNode(shared)
}

func implicitPointerLiterals() {
	builder := []*Node{{Value: 1}}
	_ = im.FreezeSlice(builder)
}

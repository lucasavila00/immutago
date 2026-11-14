package restricted_boundaries

import "github.com/lucasavila00/immutago/pkg/im"

type Local struct {
	im.LocalRef
	Value int
}
type Node struct {
	im.Shared
	Value int
}

func localPointerContainers() {
	var nilSlice []*Local
	emptySlice := []*Local{}
	emptyMap := make(map[int]*Local)
	_ = nilSlice
	_ = emptySlice
	_ = emptyMap
}
func boxedRange() {
	frozen := im.FreezeSlice([]Node{{Value: 1}})
	var value any
	for _, value = range frozen.All() {
		_ = value
	}
}

type Reader struct{}

func (Reader) Read(value *Node) int { return value.Value }
func (Reader) Make() *Node          { return &Node{} }

type Plain[T any] struct{ Value T }

func (Plain[T]) ProtectedBody() int {
	value := Node{Value: 1}
	return value.Value
}

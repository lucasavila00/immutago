package immutable_objects

import "github.com/lucasavila00/immutago/pkg/im"

type Node struct {
	im.Immut
	Value int
	Next  *Node
}
type Box[T ~int | ~string] struct {
	im.Immut
	Value T
	Next  *Box[T]
}
type Holder struct {
	im.Immut
	Node  *Node
	Items im.FrozenSlice[*Node]
}

var saved *Node

func makeNode(next *Node) *Node { return &Node{Value: 1, Next: next} }
func share(node *Node) *Node    { saved = node; return node }
func build() Holder {
	first := new(Node)
	second := makeNode(first)
	copied := second
	second = &Node{Value: 2}
	var empty *Node
	empty = copied
	_ = share(empty)
	builder := []*Node{first, copied, second}
	frozen := im.FreezeSlice(builder)
	boxed := &Box[int]{Value: 3}
	_ = boxed.Next
	return Holder{Node: copied, Items: frozen}
}

func slots() im.FrozenSlice[*Node] {
	builder := []*Node{{Value: 1}}
	alias := builder[0]
	builder[0] = &Node{Value: 2}
	_ = alias.Value
	frozen := im.FreezeSlice(builder)
	cloned := frozen.Clone()
	cloned[0] = &Node{Value: 3}
	result := im.FreezeSlice(cloned)
	return result
}
func pointerMap() im.FrozenMap[*Node, *Node] {
	key := &Node{Value: 1}
	builder := map[*Node]*Node{key: {Value: 2}}
	frozen := im.FreezeMap(builder)
	value, found := frozen.Get(key)
	_ = value
	_ = found
	return frozen
}

func allocatedCopies() *Node {
	value := Node{Value: 1}
	pointer := new(value)
	value = Node{Value: 2}
	_ = new(Node{Value: 3})
	return pointer
}

func capturedRead() func() int {
	shared := &Node{Value: 1}
	return func() int { return shared.Value }
}

func readNode(node *Node) int { return node.Value }
func scheduledReads() {
	shared := &Node{Value: 1}
	go readNode(shared)
	defer readNode(shared)
}

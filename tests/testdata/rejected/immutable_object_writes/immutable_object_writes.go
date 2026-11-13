package immutable_object_writes

import "github.com/lucasavila00/immutago/pkg/im"

type Node struct {
	im.Immut
	Value int
	Next  *Node
}

func aliases() {
	first := &Node{Value: 1}
	alias := first
	first = &Node{Value: 2}
	alias.Value = 3
	*alias = Node{Value: 4}
}
func stored() {
	node := &Node{Next: &Node{Value: 1}}
	node.Next.Value++
	builder := []*Node{&Node{Value: 2}}
	builder[0].Value = 3
	frozen := im.FreezeSlice(builder)
	shared := frozen.At(0)
	shared.Value += 4
}
func copied() {
	first := Node{Value: 1}
	copy := first
	copy.Value = 2
}

var current Node

func globalField() { current.Value = 2 }

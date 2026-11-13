package model

import "github.com/lucasavila00/immutago/pkg/im"

type MarkerAlias = im.Immut
type Number int
type Title = string

type Leaf struct {
	MarkerAlias
	Name     Title
	Count    Number
	Flags    [3]bool
	Small    int8
	Unsigned uint64
	Address  uintptr
	Float    float64
	Complex  complex128
}

type Tree struct {
	im.Immut
	Leaf     Leaf
	Children im.FrozenSlice[Tree]
	Index    im.FrozenMap[string, Tree]
}

type CountBox struct {
	im.Immut
	Value int
}
type Alias = Leaf
type Derived Leaf

func Value() Tree {
	children := im.FreezeSlice([]Tree{{Leaf: Leaf{Name: "child"}}})
	index := im.FreezeMap(map[string]Tree{})
	return Tree{
		Leaf:     Leaf{Name: "leaf", Count: 3},
		Children: children,
		Index:    index,
	}
}

package model

import "github.com/lucasavila00/immutago/pkg/im"

type MarkerAlias = im.Marker
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
	im.Marker
	Leaf     Leaf
	Children im.FrozenSlice[Tree]
	Index    im.FrozenMap[string, Tree]
}

type Scalar interface{ ~int | ~string }
type Box[T Scalar] struct {
	im.Marker
	Value T
}
type CountBox = Box[int]
type Alias = Leaf
type Derived Leaf

func Value() Tree {
	return Tree{
		Leaf:     Leaf{Name: "leaf", Count: 3},
		Children: im.FreezeSlice([]Tree{{Leaf: Leaf{Name: "child"}}}),
		Index:    im.FreezeMap(map[string]Tree{}),
	}
}

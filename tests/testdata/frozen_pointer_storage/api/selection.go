package api

import "github.com/lucasavila00/immutago/pkg/im"

type Child struct {
	im.Frozen
	Name string
	N    int
}

type Selection struct {
	im.Frozen
	Child   Child
	Pointer *Child
	Array   [2]Child
	List    im.FrozenSlice[Child]
	Table   im.FrozenMap[string, Child]
}

type Score struct {
	im.Copied
	N int
}

type Payloads struct {
	Number  im.Cell[int]
	Text    im.Cell[string]
	String  im.Cell[im.FrozenString]
	Slice   im.Cell[im.FrozenSlice[int]]
	Map     im.Cell[im.FrozenMap[int, int]]
	Pointer im.Cell[*Child]
	Record  im.Cell[Child]
	Array   im.Cell[[2]int]
	Copied  im.Cell[Score]
}

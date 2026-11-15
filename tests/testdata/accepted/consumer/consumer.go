package consumer

import (
	"example.com/consumer/model"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Record struct {
	im.Shared
	Model   model.Tree
	Names   im.FrozenSlice[im.FrozenString]
	Lookups im.FrozenMap[[2]int, im.FrozenSlice[model.Leaf]]
	Count   model.CountBox
}

func Use() {
	value := model.Value()
	value = model.Tree{Leaf: model.Leaf{Name: im.FreezeString("replacement")}}
	_ = value.Leaf.Name
	local := value.Leaf.Flags
	local[0] = true
	nativeNames := []im.FrozenString{im.FreezeString("a"), im.FreezeString("b")}
	nativeNames = append(nativeNames, im.FreezeString("c"))
	nativeNames[0] = im.FreezeString("d")
	names := im.FreezeSlice(nativeNames)
	_ = names.At(0)
	_ = names.Len()
	copy := names.Clone()
	copy[0] = im.FreezeString("independent")
	nativeMap := map[im.FrozenString]int{im.FreezeString("a"): 1}
	nativeMap[im.FreezeString("b")] = 2
	delete(nativeMap, im.FreezeString("a"))
	mapping := im.FreezeMap(nativeMap)
	mapping.Get(im.FreezeString("b"))
	_ = mapping.Len()
	mappingCopy := mapping.Clone()
	mappingCopy[im.FreezeString("b")] = 3
	_ = Record{Model: value, Names: names}
	_ = model.CountBox{Value: 1}
	_ = model.Derived{}
	_ = im.FreezeSlice([][2]im.FrozenString{{im.FreezeString("a"), im.FreezeString("b")}})
	_ = struct {
		im.Shared
		Value int
	}{Value: 1}
}

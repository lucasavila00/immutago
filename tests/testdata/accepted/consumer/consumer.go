package consumer

import (
	"example.com/consumer/model"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Record struct {
	im.Immut
	Model   model.Tree
	Names   im.FrozenSlice[string]
	Lookups im.FrozenMap[[2]int, im.FrozenSlice[model.Leaf]]
	Count   model.CountBox
}

func Use() {
	value := model.Value()
	value = model.Tree{Leaf: model.Leaf{Name: "replacement"}}
	_ = value.Leaf.Name
	local := value.Leaf.Flags
	local[0] = true
	nativeNames := []string{"a", "b"}
	nativeNames = append(nativeNames, "c")
	nativeNames[0] = "d"
	names := im.FreezeSlice(nativeNames)
	_ = names.At(0)
	_ = names.Len()
	copy := names.Clone()
	copy[0] = "independent"
	nativeMap := map[string]int{"a": 1}
	nativeMap["b"] = 2
	delete(nativeMap, "a")
	mapping := im.FreezeMap(nativeMap)
	mapping.Get("b")
	_ = mapping.Len()
	mappingCopy := mapping.Clone()
	mappingCopy["b"] = 3
	_ = Record{Model: value, Names: names}
	_ = model.CountBox{Value: 1}
	_ = model.Derived{}
	_ = im.FreezeSlice([][2]string{{"a", "b"}})
	_ = struct {
		im.Immut
		Value int
	}{Value: 1}
}

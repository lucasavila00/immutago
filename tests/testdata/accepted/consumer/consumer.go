package consumer

import (
	"example.com/consumer/model"
	im "github.com/lucasavila00/immutago/pkg/immutable"
)

type Record struct {
	im.Marker
	Model   model.Tree
	Names   im.Slice[string]
	Lookups im.Map[[2]int, im.Slice[model.Leaf]]
	Count   model.CountBox
}

func Use() {
	value := model.Value()
	value = model.Tree{Leaf: model.Leaf{Name: "replacement"}}
	_ = value.Leaf.Name
	local := value.Leaf.Flags
	local[0] = true
	names := im.NewSlice("a", "b").Append("c").Set(0, "d")
	_ = names.At(0)
	_ = names.Len()
	copy := names.Values()
	copy[0] = "independent"
	mapping := im.NewMap(map[string]int{"a": 1}).Set("b", 2).Delete("a")
	_, _ = mapping.Get("b")
	_ = mapping.Len()
	mapping.Values()["b"] = 3
	_ = Record{Model: value, Names: names}
	_ = model.Box[string]{Value: "generic"}
	_ = model.Derived{}
	_ = im.NewSlice([2]string{"a", "b"})
	_ = struct {
		im.Marker
		Value int
	}{Value: 1}
}

package freezebuilders

import "github.com/lucasavila00/immutago/pkg/im"

type Value struct {
	im.Marker
	X int
}

func Slice() (im.FrozenSlice[int], im.FrozenSlice[int]) {
	values := make([]int, 0, 3)
	values = append(values, 1, 2)
	first := im.FreezeSlice(values)
	copy := first.Clone()
	copy[0] = 9
	second := im.FreezeSlice(copy)
	return first, second
}
func Map() (im.FrozenMap[string, int], im.FrozenMap[string, int]) {
	values := make(map[string]int)
	values["a"] = 1
	values["b"] = 2
	first := im.FreezeMap(values)
	copy := first.Clone()
	copy["a"] = 9
	delete(copy, "b")
	second := im.FreezeMap(copy)
	return first, second
}
func MarkedSlice() im.FrozenSlice[Value] {
	values := []Value{{X: 1}}
	values[0].X = 2
	return im.FreezeSlice(values)
}
func MarkedMap() im.FrozenMap[string, Value] {
	values := map[string]Value{"a": {X: 1}}
	copy := values["a"]
	copy.X = 2
	values["a"] = copy
	return im.FreezeMap(values)
}
func Branch(condition bool) im.FrozenSlice[int] {
	values := []int{1}
	if condition {
		values[0] = 2
	} else {
		values[0] = 3
	}
	return im.FreezeSlice(values)
}
func Loop(count int) int {
	total := 0
	for i := 0; i < count; i++ {
		values := []int{i}
		frozen := im.FreezeSlice(values)
		total += frozen.At(0)
	}
	return total
}
func Nil() im.FrozenSlice[int] { var values []int; return im.FreezeSlice(values) }

var Global = im.FreezeSlice([]int{3})

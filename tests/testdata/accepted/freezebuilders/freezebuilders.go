package freezebuilders

import "github.com/lucasavila00/immutago/pkg/im"

type Value struct {
	im.Shared
	X int
}

type Slices struct {
	im.Shared
	First, Second im.FrozenSlice[int]
}
type Maps struct {
	im.Shared
	First, Second im.FrozenMap[string, int]
}

func Slice() Slices {
	values := make([]int, 0, 3)
	values = append(values, 1, 2)
	first := im.FreezeSlice(values)
	copy := first.Clone()
	copy[0] = 9
	second := im.FreezeSlice(copy)
	return Slices{First: first, Second: second}
}
func Map() Maps {
	values := make(map[string]int)
	values["a"] = 1
	values["b"] = 2
	first := im.FreezeMap(values)
	copy := first.Clone()
	copy["a"] = 9
	delete(copy, "b")
	second := im.FreezeMap(copy)
	return Maps{First: first, Second: second}
}
func MarkedSlice() im.FrozenSlice[Value] {
	values := []Value{{X: 1}}
	values[0] = Value{X: 2}
	frozen := im.FreezeSlice(values)
	return frozen
}
func MarkedMap() im.FrozenMap[string, Value] {
	values := map[string]Value{"a": {X: 1}}
	copy := values["a"]
	copy = Value{X: 2}
	values["a"] = copy
	frozen := im.FreezeMap(values)
	return frozen
}
func Branch(condition bool) im.FrozenSlice[int] {
	values := []int{1}
	if condition {
		values[0] = 2
	} else {
		values[0] = 3
	}
	frozen := im.FreezeSlice(values)
	return frozen
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
func Nil() im.FrozenSlice[int] { var values []int; frozen := im.FreezeSlice(values); return frozen }

var Global = im.FreezeSlice([]int{3})

func AppendMarked() im.FrozenSlice[Value] {
	var values []Value
	values = append(values, Value{X: 1})
	_ = len(values)
	_ = cap(values)
	frozen := im.FreezeSlice(values)
	return frozen
}

func CopyMarked() im.FrozenSlice[Value] {
	values := make([]Value, 1)
	source := []Value{{X: 4}}
	copy(values, source)
	frozen := im.FreezeSlice(values)
	return frozen
}

func ClearMarkedMap() im.FrozenMap[Value, Value] {
	values := make(map[Value]Value)
	values[Value{X: 1}] = Value{X: 2}
	delete(values, Value{X: 1})
	values[Value{X: 2}] = Value{X: 3}
	clear(values)
	values[Value{X: 3}] = Value{X: 4}
	frozen := im.FreezeMap(values)
	return frozen
}

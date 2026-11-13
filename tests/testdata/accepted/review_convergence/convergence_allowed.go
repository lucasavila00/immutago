package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

type Value struct {
	im.Marker
	X int
}

func sliceReads() im.FrozenSlice[Value] {
	frozen := im.FreezeSlice([]Value{{X: 1}})
	p := &frozen
	_ = (p.Len)()
	_ = (p.At)(0)
	for _, value := range (p.All)() {
		_ = value.X
	}
	builder := (p.Clone)()
	builder[0] = Value{X: 2}
	updated := im.FreezeSlice(builder)
	return updated
}

func mapReads() im.FrozenMap[int, Value] {
	frozen := im.FreezeMap(map[int]Value{1: {X: 1}})
	p := &frozen
	value, found := (p.Get)(1)
	_ = value
	_ = found
	for key, value := range (p.All)() {
		_ = key
		_ = value.X
	}
	builder := (p.Clone)()
	builder[1] = Value{X: 2}
	updated := im.FreezeMap(builder)
	return updated
}

func read(p *Value) int {
	if p == (nil) {
		return 0
	}
	return p.X
}

func singleBindings() {
	value := Value{X: 1}
	a := value.X + 1
	var b = read(&value)
	_ = a
	_ = b
	builder := []int{1}
	n := len(builder)
	_ = n
	_ = im.FreezeSlice(builder)
	for value.X = range []int{2} {
	}
	nil := &value
	_ = read(nil)
}

func scalarCallback(factory func() int) int { return factory() }

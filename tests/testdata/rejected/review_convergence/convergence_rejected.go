package review_convergence

import "github.com/lucasavila00/immutago/pkg/im"

type Value struct {
	im.Marker
	X, Y int
}

func read(p *Value) int { return p.X }

func binaryBindings() {
	var value Value
	a, b := value.X+1, 2
	_, _ = a, b
	var c, d = value.X + 1, 2
	_, _ = c, d
}

func callBindings() {
	var value Value
	a, b := read(&value), 2
	_, _ = a, b
	var c, d = read(&value), 2
	_, _ = c, d
}

func lengthBindings() {
	builder := []int{1}
	a, b := len(builder), 2
	_, _ = a, b
	var c, d = len(builder), 2
	_, _ = c, d
	_ = im.FreezeSlice(builder)
}

func rangeTargets() {
	var value Value
	for value.X, value.Y = range []int{1} {
	}
}

func methodExpressions() {
	frozen := im.FreezeMap(map[int]Value{})
	value, found := (im.FrozenMap[int, Value].Get)(frozen, 1)
	_ = value
	_ = found
}

func nilArgument() {
	_ = read((nil))
}

func savedIterator() {
	frozen := im.FreezeSlice([]Value{{X: 1}})
	iterator := ((frozen.All)())
	for _, value := range iterator {
		_ = value.X
	}
}

func callbackResult(factory func() Value) Value { return factory() }

type Factory interface{ Make() Value }

func interfaceResult(factory Factory) Value { return factory.Make() }

func nestedCallbackResult(factory func() func() Value) { _ = factory() }

func blankRangeTarget() {
	var value Value
	for _, value.X = range []int{1} {
	}
}

func nestedRangeTarget() {
	var value Value
	var a [1]int
	var n int
	for a[value.X], n = range []int{2} {
	}
	_ = a
	_ = n
}

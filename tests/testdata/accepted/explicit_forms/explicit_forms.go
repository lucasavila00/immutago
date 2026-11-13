package explicit_forms

import "github.com/lucasavila00/immutago/pkg/im"

type Good struct {
	im.Immut
	X int
}
type Alias = Good

func read(value *Alias) int { return value.X }
func fixed() Good {
	var value Good
	var pointer *Alias = (&Alias{})
	_ = read((pointer))
	value = Good{X: 1}
	return value
}
func selectBody(ready <-chan int) im.FrozenSlice[bool] {
	values := []bool{false}
	select {
	case <-ready:
		values[0] = true
	default:
		values[0] = false
	}
	frozen := im.FreezeSlice(values)
	return frozen
}
func selectValueBody(ready <-chan int) Good {
	var value Good
	select {
	case <-ready:
		value = Good{X: 1}
	default:
		value = Good{X: 2}
	}
	return value
}

type Parent struct {
	im.Immut
	Child Good
	Other int
}

func referenceReads() Parent {
	var value Parent
	p := &Good{}
	q := p
	_ = p
	_ = p.X
	_ = p == q
	_ = p != nil
	copy := *p
	copy = Good{X: 3}
	value = Parent{Child: *p, Other: copy.X}
	return value
}
func declaredGet() Good {
	frozen := im.FreezeMap(map[string]Good{"first": {X: 1}})
	var value, ok = frozen.Get("first")
	_ = ok
	value = Good{X: 2}
	return value
}

func valueSwitch() Good {
	value := Good{X: 1}
	switch value.X {
	case 1:
		value = Good{X: 2}
	default:
		value = Good{X: 3}
	}
	return value
}
func arrayRange() Good {
	values := [2]Good{{X: 1}, {X: 2}}
	var result Good
	for _, value := range values {
		result = Good{X: result.X + value.X}
	}
	return result
}

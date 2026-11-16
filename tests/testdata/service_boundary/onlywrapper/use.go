package onlywrapper

import "example.com/consumer/sealedapi"

type Alias = sealedapi.Handle
type Holder struct{ Values [1]Alias }

var Saved sealedapi.Handle
var savedHolder Holder

func keep[T any](value T) T { return value }

func Store(value sealedapi.Handle) any {
	Saved = value
	savedHolder.Values[0] = value
	values := []Holder{savedHolder}
	read := func() Alias { return keep(values[0].Values[0]) }
	return keep(read)()
}

func consume(Alias)

func Call(value Alias) { consume(value) }

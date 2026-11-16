package wrapper

import "example.com/consumer/api"

type Alias = api.DefinedRef
type Holder struct{ Values [1]Alias }

var saved Holder

func keep[T any](value T) T { return value }

func Use(ref Alias) any {
	saved.Values[0] = ref
	values := []Holder{saved}
	read := func() Alias { return keep(values[0].Values[0]) }
	return keep(read)()
}

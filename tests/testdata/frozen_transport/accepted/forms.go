package accepted

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

func identity[T any](value T) T { return value }
func body[T any](value T, data api.Data) T {
	var boxed any = data
	copy := boxed.(api.Data)
	callback := func() int { return copy.Text.Len() }
	_ = callback()
	return value
}
func readConstraint[T api.Reader](value T) int         { return value.Number() }
func constrained[T api.Count](value T) any             { return value }
func nested[T any](value T) func() T                   { return api.Capture(value) }
func frozenCall(value im.FrozenString) im.FrozenString { return value }
func countPair(value api.Count) (api.Count, bool)      { return value, true }
func takeReader(value api.Reader) int                  { return value.Number() }
func box(value any) any                                { return value }
func tupleBox(value api.Count) (any, bool)             { return countPair(value) }
func variadic(values ...any) int                       { return len(values) }

func Exercise() int {
	text := im.FreezeString("hello")
	list := im.FreezeSlice([]int{7, 8})
	table := im.FreezeMap(map[string]int{"n": 3})
	count := api.Count{N: 2}
	var reader api.Reader = count
	data := api.Alias{Text: text, Counts: [2]api.Count{count}, List: list, Map: table, Reader: reader}
	var boxed any = text
	recovered := boxed.(im.FrozenString)
	recovered, ok := boxed.(im.FrozenString)
	if !ok {
		return -1
	}
	boxed = data
	fromBox := boxed.(api.Data)
	switch value := boxed.(type) {
	case api.Data:
		fromBox = value
	}
	fromGeneric := identity(text)
	imported := api.Identity(data)
	callback := frozenCall
	result := callback(text)
	saved := api.Apply(text, func(value im.FrozenString) im.FrozenString { return value })
	capture := func() int { return text.Len() + data.Counts[0].Number() + reader.Number() }
	global := func() int { return api.Current.Text.Len() }
	_ = global
	_ = body(count, data)
	_ = constrained(count)
	_ = readConstraint(count)
	_ = nested(data)()
	_ = api.Identity(list)
	_ = api.Identity(table)
	_ = api.Identity(reader)
	_ = api.Identity(data.Counts)
	_ = box(list).(im.FrozenSlice[int])
	_ = box(table).(im.FrozenMap[string, int])
	_ = box(reader)
	_ = []any{text, data, reader, callback}
	_ = [2]any{text, count}
	_ = map[any]any{count: data}
	_ = struct{ Value any }{data}
	_ = variadic(text, count, reader)
	method := count.Number
	expression := api.Count.Number
	length := text.Len
	lengthExpression := im.FrozenString.Len
	readerMethod := reader.Number
	readerExpression := api.Reader.Number
	pair := countPair
	first, found := pair(count)
	var another, yes = pair(count)
	first, another = another, first
	_, _ = found, yes
	boxed, found = pair(count)
	_, _ = tupleBox(count)
	_ = takeReader(api.Identity(count))
	go frozenCall(text)
	defer frozenCall(text)
	return recovered.Len() + fromGeneric.Len() + result.Len() + saved.Len() +
		fromBox.Text.Len() + imported.Text.Len() + capture() + method() + expression(count) +
		length() + lengthExpression(text) + readerMethod() + readerExpression(reader) + first.N
}

func pointers(value *api.Count, reader api.Reader) int {
	var boxed any = value
	_ = boxed.(*api.Count)
	_ = api.Identity(value)
	_ = api.Capture(value)
	_ = reader.(*api.Count)
	switch reader.(type) {
	case api.Count, *api.Count:
	}
	return value.N
}

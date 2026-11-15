package implementation

import "example.com/consumer/api"

func (t *Tree) Sum(values ...int) int {
	result := int(t.Weight)
	for _, value := range values {
		result += value
	}
	return result
}
func (t *Tree) Add(prefix int, values ...api.AmountAlias) int {
	result := prefix
	for _, value := range values {
		result += int(value)
	}
	return result
}
func (t *Tree) Call(callbacks ...func() int) int {
	result := 0
	for _, callback := range callbacks {
		result += callback()
	}
	return result
}
func (t *Tree) Box(values ...any) int { return len(values) }

var SavedNumbers []int

func (t *Tree) Retain(values ...int) { SavedNumbers = values }

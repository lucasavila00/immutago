package accepted

import "github.com/lucasavila00/immutago/pkg/im"

type FreezeCache struct {
	Values im.FrozenSlice[int]
	Index  im.FrozenMap[string, int]
}

func (c *FreezeCache) Reset(input []int) {
	values := make([]int, 0, len(input))
	for _, value := range input {
		values = append(values, value)
	}
	c.Values = im.FreezeSlice(values)
	index := map[string]int{"first": c.Values.At(0)}
	c.Index = im.FreezeMap(index)
}

type SliceAlias = im.FrozenSlice[int]

// FreezeSlots uses each destination and reads a builder index before consuming it.
func FreezeSlots() int {
	var cache FreezeCache
	cache.Reset([]int{1, 2})
	var array [1]SliceAlias
	array[0] = im.FreezeSlice([]int{3})
	slices := make([]SliceAlias, 1)
	values := []int{4}
	slices[len(values)-1] = im.FreezeSlice(values)
	entries := make(map[string]im.FrozenMap[string, int])
	entries["saved"] = im.FreezeMap(map[string]int{"n": 5})
	pointer := new(SliceAlias)
	(*pointer) = (im.FreezeSlice([]int{6}))
	first, _ := cache.Index.Get("first")
	saved, _ := entries["saved"].Get("n")
	return cache.Values.At(1) + first + array[0].At(0) + slices[0].At(0) + saved + pointer.At(0)
}

// FreezeInterfaces keeps the same assignment rules for compatible interface slots.
func FreezeInterfaces() {
	var boxed any = im.FreezeSlice([]int{1})
	boxed = im.FreezeSlice([]int{2})
	holder := struct{ Value any }{}
	holder.Value = im.FreezeMap(map[int]int{1: 2})
	values := make([]any, 1)
	values[0] = im.FreezeSlice([]int{3})
	slot := &boxed
	*slot = im.FreezeSlice([]int{4})
	_ = holder
}

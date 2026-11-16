package rejected

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

type FreezeCache struct{ Values im.FrozenSlice[int] }

func freezeProtectedField(root *api.Data) {
	root.List = im.FreezeSlice([]int{1})
	root.Map = im.FreezeMap(map[string]int{"n": 1})
}

func freezeSlotAlias(cache *FreezeCache) {
	values := []int{1}
	alias := values
	_ = alias
	cache.Values = im.FreezeSlice(values)
}

func freezeSlotConsumed(cache *FreezeCache) {
	values := []int{1}
	cache.Values = im.FreezeSlice(values)
	values[0] = 2
}

func freezeSlotMultiple(cache *FreezeCache) {
	cache.Values, cache.Values = im.FreezeSlice([]int{1}), im.FreezeSlice([]int{2})
}

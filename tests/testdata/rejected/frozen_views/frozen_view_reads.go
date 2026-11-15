package localviews

import "github.com/lucasavila00/immutago/pkg/im"

// SharedElements cannot leave a local view through At or Get.
func SharedElements() {
	shared := im.FreezeSlice([]im.FrozenString{im.FreezeString("text")})
	_ = shared.ByValue()
	pointers := im.FreezeSlice([]*Shared{&Shared{}})
	_ = pointers.ByValue()
	_ = im.FreezeByValueMap(map[int]Retained{1: {}})
}

// StringKeys supports lookup and copying, but cannot yield the stored key.
func StringKeys() {
	key := im.FreezeString("key")
	values := im.FreezeByValueMap(map[im.FrozenString]int{key: 1})
	for key, value := range values.All() {
		_ = key
		_ = value
	}
}

// Iterator cannot hide a local collection in a scalar-only closure.
func Iterator() {
	values := im.FreezeByValueSlice([]int{1})
	all := values.All()
	_ = all
	mapping := im.FreezeByValueMap(map[int]int{1: 2})
	pairs := mapping.All()
	_ = pairs
}

// NestedClone would copy headers that still retain local storage.
func NestedClone(text Text) {
	values := im.FreezeByValueSlice([]Text{text})
	copy := values.Clone()
	_ = copy
	mapping := im.FreezeByValueMap(map[int]Text{1: text})
	other := mapping.Clone()
	_ = other
}

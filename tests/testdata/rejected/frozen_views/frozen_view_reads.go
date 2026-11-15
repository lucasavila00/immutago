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

type RetainedKey struct {
	im.Shared
	Text im.FrozenString
}

// Keys permits only the exact string wrapper, not arrays or containing records.
func Keys() {
	_ = im.FreezeByValueMap(map[[1]im.FrozenString]int{})
	_ = im.FreezeByValueMap(map[[0]im.FrozenString]int{})
	_ = im.FreezeByValueMap(map[RetainedKey]int{})
}

// SharedPoint is valid inside a ByValue result, but not as the result itself.
func (*Service) SharedPoint() Shared    { return Shared{} }
func (*Service) SharedArray() [1]Shared { return [1]Shared{} }

// LocalKeyClone cannot keep the local key through a shallow native map copy.
func LocalKeyClone(key Text) {
	values := im.FreezeByValueMap(map[Text]int{key: 1})
	copy := values.Clone()
	_ = copy
}

package docrejected

import "github.com/lucasavila00/immutago/pkg/im"

type nilLocal struct{ im.LocalRef }

// shadowedNil is an existing handle, not Go's predeclared nil value.
func shadowedNil() {
	nil := []int{1}
	values := []int(nil)
	_ = im.FreezeSlice(values)
}

// shadowedMapNil cannot become exclusive storage through a conversion.
func shadowedMapNil() {
	nil := map[string]int{"a": 1}
	_ = im.FreezeMap(map[string]int(nil))
}

// convertedHandle cannot copy a handle even when that handle started as nil.
func convertedHandle() {
	values := []int(nil)
	alias := []int(values)
	_ = im.FreezeSlice(alias)
}

// convertedDirectHandle must not hide an alias at the Freeze boundary.
func convertedDirectHandle() {
	values := map[string]int(nil)
	_ = im.FreezeMap(map[string]int(values))
}

// consumedNilBuilder obeys the same use-after-Freeze rule as allocated builders.
func consumedNilBuilder() {
	values := []int(nil)
	_ = im.FreezeSlice(values)
	values = append(values, 1)
}

// nestedNilConversion is not the single conversion of built-in nil that is allowed.
func nestedNilConversion() {
	_ = im.FreezeSlice([]int([]int(nil)))
}

// localPointerNilBuilder cannot store LocalRef pointers, even in an empty container.
func localPointerNilBuilder() {
	values := []*nilLocal(nil)
	_ = im.FreezeSlice(values)
}

// mutableElementNilBuilder must still prove its element type immutable.
func mutableElementNilBuilder() {
	_ = im.FreezeSlice([][]int(nil))
}

var existingNilSource = []int{1}

// nilArgumentSource returns existing storage instead of constructing a nil handle.
func nilArgumentSource(any) []int { return existingNilSource }

// nilArgumentCall is a function call, not a conversion of nil.
func nilArgumentCall() {
	_ = im.FreezeSlice(nilArgumentSource(nil))
}

type nilShared struct{ im.Shared }

// convertedProtectedHandle cannot turn a protected handle copy into fresh storage.
func convertedProtectedHandle() {
	values := []nilShared(nil)
	alias := []nilShared(values)
	_ = im.FreezeSlice(alias)
}

// nestedNilFreeze must still bind the frozen result before passing it to a call.
func nestedNilFreeze() {
	readNilFrozen(im.FreezeSlice([]nilShared(nil)))
}

// readNilFrozen reads an immutable wrapper by value.
func readNilFrozen(values im.FrozenSlice[nilShared]) { _ = values.Len() }

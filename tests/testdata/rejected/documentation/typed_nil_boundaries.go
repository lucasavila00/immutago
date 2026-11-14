package docrejected

import "github.com/lucasavila00/immutago/pkg/im"

type nilInts []int
type nilIntsAlias = nilInts
type nilShared struct{ im.Shared }

// convertedNilDeclarations must use a zero declaration or bare nil initializer.
func convertedNilDeclarations() {
	values := []int(nil)
	_ = im.FreezeSlice(values)
	var scores = map[string]int(nil)
	_ = im.FreezeMap(scores)
	var named nilInts = nilInts(nil)
	_ = im.FreezeSlice(named)
	alias := (nilIntsAlias)((nil))
	_ = im.FreezeSlice(alias)
	objects := []nilShared(nil)
	_ = im.FreezeSlice(objects)
}

// convertedNilInputs must pass bare nil with explicit type arguments to Freeze.
func convertedNilInputs() {
	_ = im.FreezeSlice([]int(nil))
	_ = im.FreezeMap(map[string]int(nil))
	_ = im.FreezeSlice(nilInts(nil))
	_ = im.FreezeSlice((nilIntsAlias)((nil)))
	_ = im.FreezeSlice([]nilShared(nil))
}

// shadowedNil is an existing handle, not Go's predeclared nil value.
func shadowedNil() {
	nil := []int{1}
	var values []int = nil
	_ = im.FreezeSlice(values)
}

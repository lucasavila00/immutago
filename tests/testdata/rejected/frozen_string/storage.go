package frozenstring

import "github.com/lucasavila00/immutago/pkg/im"

type Text string
type Alias = string

type Shared struct {
	im.Shared
	Native string
}

type Scoped struct {
	im.Scoped
	Named Text
	Alias Alias
	Array [0]string
}

var NativeSlice im.FrozenSlice[string]
var NativeKey im.FrozenMap[string, int]
var NativeValue im.FrozenMap[int, string]

func NativeBuilder() {
	values := []string{"text"}
	_ = im.FreezeSlice(values)
}

type Unmarked struct{ Text im.FrozenString }

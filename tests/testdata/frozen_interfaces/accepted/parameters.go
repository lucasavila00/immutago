package accepted

import (
	"example.com/consumer/api"
	"example.com/consumer/implementation"
	"github.com/lucasavila00/immutago/pkg/im"
)

type Number int

func (n Number) Number() int       { return int(n) }
func identity[T any](v T) T        { return v }
func inputPair() (any, func() int) { return 4, func() int { return 5 } }

// InputCalls uses concrete methods, interface methods, and generated methods.
func InputCalls() [6]int {
	tree := &implementation.Tree{Weight: 3}
	var reader api.InputReader = tree
	local := api.NewInputReaderProjection(im.ProjectionOf(reader))
	var value api.ReaderAlias = Number(4)
	var callback api.CallbackAlias = identity(func() int { return 5 })
	return [6]int{
		tree.Apply(inputPair()),
		reader.Apply(inputPair()),
		local.Apply(inputPair()),
		reader.Named(value, callback),
		local.Aliased(value, callback),
		local.ReadLocal(im.ProjectString("unchanged")),
	}
}

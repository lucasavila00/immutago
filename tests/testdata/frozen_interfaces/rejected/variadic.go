package rejected

import (
	"example.com/consumer/api"
	"example.com/consumer/implementation"
	"github.com/lucasavila00/immutago/pkg/im"
)

func variadicInputs(reader api.VariadicReader, tree *implementation.Tree, text im.StringProjection) {
	local := api.NewVariadicReaderProjection(im.ProjectionOf(reader))
	_ = reader.Box(text)
	_ = tree.Box(text)
	_ = local.Box(text)
	_ = reader.Call(func() int { return text.Len() })
	_ = tree.Call(func() int { return text.Len() })
	_ = local.Call(func() int { return text.Len() })
	_ = reader.Box(protectedInput)
	_ = local.Box(protectedInput)
}

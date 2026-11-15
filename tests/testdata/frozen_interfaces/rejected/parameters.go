package rejected

import (
	"example.com/consumer/api"
	"example.com/consumer/implementation"
	"github.com/lucasavila00/immutago/pkg/im"
)

func inputBoxing(reader api.InputReader, tree *implementation.Tree, text im.StringProjection) {
	local := api.NewInputReaderProjection(im.ProjectionOf(reader))
	_ = reader.Apply(text, nil)
	_ = tree.Apply(text, nil)
	_ = local.Apply(text, nil)
	_ = local.Apply([1]im.StringProjection{text}, nil)
	_ = reader.Named(implementation.Count{N: 1}, nil)
	_ = local.Aliased(implementation.Count{N: 1}, nil)
	_ = local.Apply(localPair())
}

func localPair() (im.StringProjection, func() int) {
	return im.ProjectString("x"), nil
}

func protectedInput() im.StringProjection { return im.ProjectString("x") }

func inputCallbacks(reader api.InputReader, text im.StringProjection) {
	local := api.NewInputReaderProjection(im.ProjectionOf(reader))
	_ = reader.Apply(nil, func() int { return text.Len() })
	_ = local.Apply(nil, func() int { return text.Len() })
	_ = reader.Apply(protectedInput, nil)
	_ = local.Apply(protectedInput, nil)
	_ = reader.Apply(identity(text), nil)
}

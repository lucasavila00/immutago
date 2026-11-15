package accepted

import "github.com/lucasavila00/immutago/pkg/im"

type TextProjection = im.Projection[Label]

func projectionAdapters(text TextProjection, frozen im.Projection[im.FrozenString],
	sequence im.Projection[im.FrozenSlice[int]], mapping im.Projection[im.FrozenMap[string, int]]) int {
	_ = im.StringOf(text).Is("label")
	_ = (im.StringOf[Label])(text).Equal(im.ProjectString(Label("label")))
	_ = im.StringOfFrozen(frozen).Is("frozen")
	_ = im.SliceOf(sequence).Len()
	_ = im.MapOf(mapping).Len()
	return im.Value(im.ProjectionOf(3))
}

package rejected

import "github.com/lucasavila00/immutago/pkg/im"

func projectionAdapters(text im.Projection[string], frozen im.Projection[im.FrozenString],
	sequence im.Projection[im.FrozenSlice[int]], mapping im.Projection[im.FrozenMap[string, int]]) {
	stringOf := im.StringOf[string]
	_ = stringOf(text)
	go im.StringOf(text)
	defer im.StringOfFrozen(frozen)
	_ = func() int { return im.SliceOf(sequence).Len() }
	var boxed any = im.MapOf(mapping)
	_ = boxed
	projection := im.StringOf(text)
	method := projection.Is
	_ = method
	_ = im.StringProjection.Is(projection, "text")
}

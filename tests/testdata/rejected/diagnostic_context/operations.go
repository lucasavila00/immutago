package diagnosticcontext

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import "github.com/lucasavila00/immutago/pkg/im"

type Scorer interface {
	im.FrozenInterface
	Score() int
}

type Settings struct {
	im.Frozen
	Limit int
}

type NamedText string
type Text = string
type FrozenText = im.FrozenString
type Record = Settings

func valueReads(title im.Projection[string], named im.Projection[NamedText], alias im.Projection[Text],
	label im.Projection[im.FrozenString], frozenAlias im.Projection[FrozenText], record im.Projection[Record]) {
	_ = im.Value(title)
	_ = im.Value[NamedText](named)
	_ = im.Value[Text](alias)
	_ = im.Value(label)
	_ = im.Value[FrozenText](frozenAlias)
	_ = im.Value(record)
}

func collectionReads(slice im.Projection[im.FrozenSlice[string]], lookup im.Projection[im.FrozenMap[string, int]],
	array im.Projection[[2]string]) {
	_ = im.Value(slice)
	_ = im.Value(lookup)
	_ = im.Value(array)
}

func recoverScorer(boxed any) { _ = boxed.(Scorer) }

func recoverScorerPair(boxed any) {
	value, ok := boxed.(Scorer)
	_, _ = value, ok
}

func cloneText(values im.SliceProjection[string], alias im.SliceProjection[Text],
	frozen im.SliceProjection[im.FrozenString], records im.SliceProjection[Settings]) {
	_ = values.Clone()
	_ = alias.Clone()
	_ = frozen.Clone()
	_ = records.Clone()
}

func cloneMap(values im.MapProjection[string, string], nested im.SliceProjection[im.FrozenSlice[int]]) {
	_ = values.Clone()
	_ = nested.Clone()
}

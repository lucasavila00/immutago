package diagnosticcontext

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Scorer interface {
	im.FrozenInterface
	Score() int
}

type Settings struct {
	im.Frozen
	Limit int
}

func (s *Settings) Score() int { return s.Limit }

type Score struct {
	im.Copied
	Value int
}

type Reply struct {
	im.Local
	Name im.StringProjection
}

type Summary struct {
	im.Local
	Count int
	Reply Reply
}

type NamedText string
type Text = string
type FrozenText = im.FrozenString

func textReads(title im.Projection[string], named im.Projection[NamedText], alias im.Projection[Text],
	label im.Projection[im.FrozenString], frozenAlias im.Projection[FrozenText]) {
	_ = im.StringOf(title).Unsafe()
	_ = im.StringOf(title).Clone()
	_ = im.StringOf(named).Unsafe()
	_ = im.StringOf(alias).Clone()
	_ = im.StringOfFrozen(label).Unsafe()
	_ = im.StringOfFrozen(frozenAlias).Clone()
}

func recordReads(record im.Projection[Settings], pointer im.Projection[*Settings], scorer im.Projection[Scorer]) {
	_ = record.Unsafe()
	// CopyRecord makes a new root for the generated pointer projection.
	_ = NewSettingsProjection(im.CopyRecord(record)).Limit()
	_ = NewSettingsProjection(pointer).Limit()
	_ = NewScorerProjection(scorer).Score()
}

func collectionReads(slice im.Projection[im.FrozenSlice[string]], lookup im.Projection[im.FrozenMap[string, int]],
	array im.Projection[[2]string]) {
	_ = im.SliceOf(slice).Len()
	_ = im.MapOf(lookup).Len()
	_ = array.Unsafe()
}

func recoverScorer(boxed any) Scorer {
	tree := boxed.(*Settings)
	return Scorer(tree)
}

func recoverScorerPair(boxed any) (Scorer, bool) {
	tree, ok := boxed.(*Settings)
	return Scorer(tree), ok
}

func localText(text string, frozen im.FrozenString, projected im.StringProjection) Summary {
	first := Reply{Name: im.ProjectString(text)}
	second := Reply{Name: frozen.Projection()}
	third := Reply{Name: projected}
	_ = first
	_ = second
	return Summary{Count: 1, Reply: third}
}

func cloneStrings(values im.SliceProjection[string]) []string {
	out := make([]string, values.Len())
	for index, value := range values.All() {
		out[index] = im.StringOf(value).Clone()
	}
	return out
}

func cloneFrozenStrings(values im.SliceProjection[im.FrozenString]) []string {
	out := make([]string, values.Len())
	for index, value := range values.All() {
		out[index] = im.StringOfFrozen(value).Clone()
	}
	return out
}

func cloneMap(values im.MapProjection[string, string]) map[string]string {
	out := make(map[string]string, values.Len())
	for key, value := range values.All() {
		out[im.StringOf(key).Clone()] = im.StringOf(value).Clone()
	}
	return out
}

func copyScores(values im.SliceProjection[Settings]) []Score {
	out := make([]Score, values.Len())
	for index, value := range values.All() {
		settings := value.Unsafe()
		out[index] = Score{Value: settings.Limit}
	}
	return out
}

func cloneNested(values im.SliceProjection[im.FrozenSlice[int]]) [][]int {
	out := make([][]int, values.Len())
	for index, value := range values.All() {
		out[index] = im.SliceOf(value).Clone()
	}
	return out
}

type SettingsReply struct {
	im.Local
	Value SettingsProjection
}

func settingsReply(value *Settings) SettingsReply {
	return SettingsReply{Value: NewSettingsProjection(im.ProjectionOf(value))}
}

type Worker struct {
	supervise.Service[supervise.NoArgs]
}

func (w *Worker) Run(ctx context.Context) { <-ctx.Done() }

func (w *Worker) Read(value *Settings) SettingsProjection {
	return NewSettingsProjection(im.ProjectionOf(value))
}

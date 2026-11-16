package diagnosticcontext

import "github.com/lucasavila00/immutago/pkg/im"

type InterfaceMarkerData struct {
	im.Local
	Markers [0]ScorerProjection
}

type InterfaceReply struct {
	im.Local
	Model ScorerProjection
}

type InterfaceSummary struct {
	im.Local
	Count  int
	Models [0]ScorerProjection
}

type NestedInterfaceReply struct {
	im.Local
	Replies [2]InterfaceReply
}

type NestedInterfaceSummary struct {
	im.Local
	Summaries [2]InterfaceSummary
}

func scorerReply(value Scorer) InterfaceReply {
	return InterfaceReply{Model: NewScorerProjection(im.ProjectionOf(value))}
}

type StoredScorer struct {
	im.Frozen
	Model Scorer
}

var currentScorer im.Cell[Scorer]

func storeScorer(value Scorer) im.FrozenSlice[Scorer] {
	currentScorer.Store(value)
	_ = NewScorerProjection(currentScorer.Load()).Score()
	values := []Scorer{value}
	frozen := im.FreezeSlice(values)
	return frozen
}

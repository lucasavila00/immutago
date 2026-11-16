package proofdiagnostics

import "github.com/lucasavila00/immutago/pkg/im"

type Plain struct{ N int }
type Counts [2]int

var payload im.Cell[any]
var detail im.Cell[Plain]
var retries im.Cell[*int]

type Summary struct {
	im.Local
	Totals Counts
}

type Formatter interface {
	Join(parts ...im.StringProjection) int
}

func labels(name im.StringProjection) {
	names := []im.StringProjection{name}
	alias := names
	_ = alias
}

func mapped(name im.StringProjection) {
	names := map[int]im.StringProjection{0: name}
	alias := names
	_ = alias
}

func (s *Score) Set(value float64) { s.Value = value }

type LocalScores struct {
	im.Local
	Score Score
}

func badCalls(value LocalScores, p *LocalScores, queue chan Point) {
	value.Score.Set(2)
	_ = (*LocalScores)(nil)
	go readLocal(p)
}

func readLocal(p *LocalScores) float64 { return p.Score.Value }

var interfaceMarker im.Cell[im.FrozenInterface]

func markerProjection(value im.FrozenInterface) {
	_ = im.ProjectionOf(value)
}

func (p *Point) ReadPointer() int { return p.X }

func pointerReceiver(value Point) int { return value.ReadPointer() }

type InterfaceMarker = im.FrozenInterface

func markerAlias(value InterfaceMarker) {
	projection := im.ProjectionOf(value)
	_ = im.Value(projection)
}

type Text = string

func cloneText(value im.SliceProjection[string])    { _ = value.Clone() }
func cloneTextAlias(value im.SliceProjection[Text]) { _ = value.Clone() }

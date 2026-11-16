package proofdiagnostics

import "github.com/lucasavila00/immutago/pkg/im"

type Failure = error

func (w *Worker) Retry() Failure      { return nil }
func (w *Worker) Best() (Score, bool) { return Score{Value: 1}, true }

// Ordinary holders can keep Frozen handles and pointers.
type Saved struct {
	Point  *Point
	Text   im.FrozenString
	Values im.FrozenSlice[int]
	Index  im.FrozenMap[int, int]
}

// A pointer alias also satisfies the Frozen payload proof.
type PointPointer = *Point

var pointCell im.Cell[PointPointer]

func pointerPayload(p PointPointer) {
	pointCell.Store(p)
	_ = im.ProjectionOf(p)
	_ = im.FreezeSlice([]PointPointer{p})
}

// A defined pointer uses ordinary storage, but is not a Frozen payload type.
type Selected *Point

type PointerHolder struct{ Point Selected }

func savePointer(p Selected) PointerHolder { return PointerHolder{Point: p} }

// copy moves elements, not the native slice handle.
func copyInputs(source, destination []int, text string) {
	alias := source
	values := make([]int, len(source))
	copy(values, alias)
	out := destination
	copy(out, values)
	_ = im.FreezeSlice(values)
	bytes := make([]byte, len(text))
	copy(bytes, text)
	_ = im.FreezeSlice(bytes)
}

// Frozen storage can use a concrete Frozen type, a Frozen interface, or a Copied record.
type Record interface{ im.FrozenInterface }

var payload im.Cell[Record]
var detail im.Cell[Score]
var retries im.Cell[int]
var pointer im.Cell[*Point]

type Summary struct {
	im.Local
	Totals [2]int
}

func label(name im.StringProjection) {
	names := []im.StringProjection{name}
	_ = names[0].Len()
}

func mapped(name im.StringProjection) {
	names := map[int]im.StringProjection{0: name}
	_ = names[0].Len()
}

func (s *Score) Set(value float64) { s.Value = value }

type LocalScores struct {
	im.Local
	Score Score
}

func goodCalls(value LocalScores, p *LocalScores, queue chan *Point) {
	score := value.Score
	score.Set(2)
	var missing *LocalScores = nil
	_ = missing
	_ = readLocal(p)
}

func readLocal(p *LocalScores) float64 { return p.Score.Value }

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

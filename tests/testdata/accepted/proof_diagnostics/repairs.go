package proofdiagnostics

//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Point struct {
	im.Frozen
	X int
}

type Score struct {
	im.Copied
	Value float64
}

type Label struct {
	im.Copied
	Size int
}

type Plain struct{ Text string }

// Local fields can use every read-result category.
type Reply struct {
	im.Local
	Score  Score
	Text   im.StringProjection
	Handle im.UncheckedValue[Plain]
	Names  [2]im.StringProjection
}

type Worker struct {
	supervise.Service[supervise.NoArgs]
}

func (w *Worker) Run(ctx context.Context) { <-ctx.Done() }
func (w *Worker) Name() im.StringProjection {
	return im.ProjectString("name")
}
func (w *Worker) Read() Reply { return Reply{} }

// Frozen elements can contain strings and pointers to Frozen records.
func names() {
	texts := im.FreezeSlice([]string{"name"})
	_ = texts
	points := []*Point{{X: 1}}
	frozen := im.FreezeSlice(points)
	_ = frozen
}

func read(ref supervise.Ref[Worker], reply Reply) {
	_ = ref.Borrow().Name()
	use(reply)
}

func use(reply Reply) {}

var counts im.Cell[int]

// Read Copied values through value projections, not pointer projections.
func record(score im.Projection[Score], values im.SliceProjection[int]) {
	_ = im.Value(score)
	_ = im.Value(counts.Load())
	_ = im.Value(values.At(0))
}

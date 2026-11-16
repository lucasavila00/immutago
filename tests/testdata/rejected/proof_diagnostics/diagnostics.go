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

// Label fails the Copied field rule, which excludes strings and reference handles.
type Label struct {
	im.Copied
	Text string
}

// Reply fails the read-result field rule.
type Reply struct {
	im.Local
	Text string
}

type Worker struct {
	supervise.Service[supervise.NoArgs]
}

func (w *Worker) Run(ctx context.Context) { <-ctx.Done() }

// Name fails the same read-result rule as a Local field.
func (w *Worker) Name() string { return "" }

// names needs Frozen elements, even when an element is a valid read result.
func names(name im.StringProjection) {
	values := []im.StringProjection{name}
	frozen := im.FreezeSlice(values)
	_ = frozen
}

// The native builder rule applies here without a Freeze call.
var points []Point

func keep(ref supervise.Ref[Worker]) {
	worker := ref.Borrow()
	_ = worker
}

type View struct {
	im.Local
	Text im.StringProjection
}

func later(view View) { go use(view) }
func use(view View)   {}

var counts im.Cell[int]

// These pointer APIs require a concrete Frozen record, not a Copied payload.
func record(score im.Projection[Score], values im.SliceProjection[int]) {
	_ = im.CopyRecord(score)
	_ = counts.LoadPointer()
	_ = values.AtPointer(0)
}

// The same requirement applies to a pointer projection without an API call.
func pointer(value im.Projection[*Score]) { _ = value }

type Name = im.StringProjection

// Aliases and nested elements still use the Frozen collection proof.
func nested(value Name) {
	values := [1]Name{value}
	frozen := im.FreezeMap(map[string][1]Name{"name": values})
	_ = frozen
}

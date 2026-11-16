package proofdiagnostics

import "github.com/lucasavila00/immutago/pkg/im"

type Failure error

type CodedFailure interface {
	error
	Code() int
}

func (w *Worker) Retry() Failure          { return nil }
func (w *Worker) RetryCode() CodedFailure { return nil }

type Saved struct {
	Point  Point
	Text   im.StringProjection
	Values im.SliceProjection[int]
	Index  im.MapProjection[int, int]
}

// A defined pointer can use ordinary slots, but cannot be a Frozen payload.
type Selected *Point

var pointCell im.Cell[Selected]

func pointerPayload(p Selected) {
	_ = im.ProjectionOf(p)
	_ = im.FreezeSlice([]Selected{p})
}

func copiedBuilder() {
	values := []int{1}
	alias := values
	_ = alias
	_ = im.FreezeSlice(values)
	_ = values[0]
}

func copyAfterFreeze(destination []int) {
	values := []int{1}
	_ = im.FreezeSlice(values)
	copy(destination, values)
}

func copyProtectedElement(source []*int) {
	values := make([]*int, len(source))
	copy(values, source)
	_ = im.FreezeSlice(values)
}

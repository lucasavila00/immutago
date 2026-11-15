package valid

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
	"testing"
)

// TestReadResults checks service, generated, and primitive handle reads.
func TestReadResults(t *testing.T) {
	root := supervise.NewRoot(nil)
	defer func() { <-root.Stop() }()
	loader, err := supervise.Start[Loader](root, supervise.NoArgs{})
	if err != nil {
		t.Fatal(err)
	}
	current := loader.Borrow().Current()
	pair, err := loader.Borrow().Read()
	if err != nil {
		t.Fatal(err)
	}
	if api.Read(current) != 1 || api.Read(pair) != 1 {
		t.Fatal("service lost the model")
	}
	reply := loader.Borrow().Serving()
	if api.Read(Capture(reply)()) != 1 {
		t.Fatal("Local result lost the model")
	}
	var cell ModelCell
	cell.Store(Model{Table: current, Models: [2][1]Handle{{current}, {current}}})
	projection := cell.Load()
	if ReadProjection(projection) != 2 || api.Read(CaptureProjection(projection)()) != 1 {
		t.Fatal("generated getter lost the model")
	}
	slices := im.FreezeSlice([]Handle{current})
	maps := im.FreezeMap(map[int]Handle{1: current})
	if ReadElements(slices.Projection(), maps.Projection()) != 2 {
		t.Fatal("collection projection lost the model")
	}
	raw := &RawHolder{}
	raw.Cell.Store(current)
	if api.Read(ReadCell(raw)) != 1 {
		t.Fatal("Cell projection lost the model")
	}
	if got := CopyArray(im.ProjectionOf([2][1]Handle{{current}, {current}})); api.Read(got[1][0]) != 1 {
		t.Fatal("array read lost the model")
	}
	_ = ZeroArray(im.ProjectionOf([0]Handle{}))
	if im.Value(im.ProjectionOf(Handle{})) != (Handle{}) {
		t.Fatal("zero handle changed")
	}
	_ = Ordinary(current)
}

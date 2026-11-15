package accepted

import (
	"example.com/consumer/api"
	"example.com/consumer/implementation"
	"github.com/lucasavila00/immutago/pkg/im"
	"testing"
)

func TestInterfaceReads(t *testing.T) {
	if got := Exercise(); got != 9 {
		t.Fatal(got)
	}
	summary := current.Load().Readers().At(0).Summary()
	if summary.Count != 3 {
		t.Fatal(summary.Count)
	}
	summary.Count++
	if current.Load().Readers().At(0).Summary().Count != 3 {
		t.Fatal("changing a copied result changed its Frozen implementation")
	}
	if implementation.Saved == nil {
		t.Fatal("method did not retain its receiver")
	}
	var empty api.ScorerProjection
	if !empty.IsNil() {
		t.Fatal("zero interface projection is not nil")
	}
	var p *implementation.Tree
	var s api.Scorer = p
	local := api.NewScorerProjection(im.ProjectionOf(s))
	if local.IsNil() {
		t.Fatal("typed nil pointer became a nil interface")
	}
}

// TestInterfaceResultBuffers checks aliasing, values, and independent Clone storage.
func TestInterfaceResultBuffers(t *testing.T) {
	reader := implementation.Make()
	buffer := make([]byte, 4)
	value, found, out, err := ReadResults(reader, buffer)
	if value != 3 || !found || len(out) != 1 || out[0] != 7 || &out[0] != &buffer[0] || err != nil {
		t.Fatal("lookup results changed")
	}
	local := api.NewReaderProjection(im.ProjectionOf(reader))
	clone := local.Clone()
	clone[0] = 9
	if local.Clone()[0] != 4 {
		t.Fatal("Clone result changed frozen data")
	}
}

// TestOrdinaryResults checks ordinary outputs and declared Frozen receiver results.
func TestOrdinaryResults(t *testing.T) {
	reader := implementation.Make()
	FrozenResults(reader)
	local := api.NewReaderProjection(im.ProjectionOf(reader))
	text, value, callback, entries, buffers, err := local.Ordinary(api.Input{Value: 5, Callback: func() int { return 6 }})
	if text != "tree" || value != 5 || callback() != 6 || entries[1] != 2 || len(buffers) != 1 || err != nil {
		t.Fatal("ordinary interface results changed")
	}
	if local.Self() != reader {
		t.Fatal("declared method changed its Frozen receiver")
	}
}

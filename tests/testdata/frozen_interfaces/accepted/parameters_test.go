package accepted

import (
	"testing"

	"example.com/consumer/api"
	"example.com/consumer/implementation"
	"github.com/lucasavila00/immutago/pkg/im"
)

// TestOrdinaryInputs checks forwarding, nil inputs, and ordinary callback retention.
func TestOrdinaryInputs(t *testing.T) {
	if got := InputCalls(); got != [7]int{12, 12, 12, 12, 12, 12, 12} {
		t.Fatal(got)
	}
	var reader api.InputReader = &implementation.Tree{Weight: 3}
	local := api.NewInputReaderProjection(im.ProjectionOf(reader))
	if local.Apply(nil, nil) != 3 {
		t.Fatal("nil inputs changed")
	}
	var value api.ReaderAlias
	var callback api.CallbackAlias
	if local.Aliased(value, callback) != 3 {
		t.Fatal("zero inputs changed")
	}
	calls := 0
	if got := local.Apply(4, func() int { calls++; return 5 }); got != 12 || calls != 1 {
		t.Fatal("callback was not called once")
	}
	if implementation.SavedInput != 4 || implementation.SavedCallback() != 5 || calls != 2 {
		t.Fatal("ordinary inputs were not retained")
	}
}

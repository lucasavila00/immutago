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

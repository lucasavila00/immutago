package valid

import (
	"testing"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// TestStartup checks the value paths accepted by the checker.
func TestStartup(t *testing.T) {
	root := supervise.NewRoot(nil)
	defer func() { <-root.Stop() }()
	if _, err := supervise.Start[Direct](root, supervise.NoArgs{}); err != nil {
		t.Fatal(err)
	}
	if _, err := supervise.Start[NestedAlias](root, supervise.NoArgs{}); err != nil {
		t.Fatal(err)
	}
	if _, err := supervise.Start[Defined](root, supervise.NoArgs{}); err != nil {
		t.Fatal(err)
	}
	if _, err := supervise.Start[Recursive](root, supervise.NoArgs{}); err != nil {
		t.Fatal(err)
	}
	if _, err := supervise.Start[Anonymous](root, supervise.NoArgs{}); err != nil {
		t.Fatal(err)
	}
}

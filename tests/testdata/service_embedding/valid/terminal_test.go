package valid

import (
	"testing"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// TestTerminalStartup checks the value paths accepted by the checker.
func TestTerminalStartup(t *testing.T) {
	root := supervise.NewRoot(nil)
	defer func() { <-root.Stop() }()
	if err := startMixed(root); err != nil {
		t.Fatal(err)
	}
	if _, err := supervise.Start[TerminalDirect](root, supervise.NoArgs{}); err != nil {
		t.Fatal(err)
	}
	if _, err := supervise.Start[TerminalNestedAlias](root, supervise.NoArgs{}); err != nil {
		t.Fatal(err)
	}
	if _, err := supervise.Start[TerminalDefined](root, supervise.NoArgs{}); err != nil {
		t.Fatal(err)
	}
	if _, err := supervise.Start[TerminalRecursive](root, supervise.NoArgs{}); err != nil {
		t.Fatal(err)
	}
	if _, err := supervise.Start[TerminalAnonymous](root, supervise.NoArgs{}); err != nil {
		t.Fatal(err)
	}
}

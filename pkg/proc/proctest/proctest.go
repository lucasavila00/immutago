// Package proctest provides test helpers for supervised processes.
package proctest

import (
	"testing"
	"time"

	"github.com/lucasavila00/immutago/pkg/proc"
)

// NewRoot returns a root supervisor that stops when t ends.
// It fails t if the tree does not stop in 5 s.
func NewRoot(t testing.TB) *proc.Supervisor {
	root := proc.NewRoot(nil)
	t.Cleanup(func() {
		select {
		case <-root.Stop():
		case <-time.After(5 * time.Second):
			t.Errorf("process tree did not stop in 5 s")
		}
	})
	return root
}

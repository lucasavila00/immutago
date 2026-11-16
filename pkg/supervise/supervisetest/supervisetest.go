// Package supervisetest gives service tests a root group with automatic cleanup.
// NewRoot registers a test cleanup that requests shutdown and waits for all
// children. Tests can use the returned group with supervise.Start and StartGroup.
package supervisetest

import (
	"sync"
	"testing"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// NewRoot reports supervised failures and their stacks with t.Errorf and registers
// cleanup with t. The first failure requests Stop before it can select a terminal
// root failure. Cleanup waits up to 5 seconds for the group to stop, then reports a test
// failure if work remains. Reports after cleanup are ignored.
// Run methods must observe cancellation. Use a subprocess with supervise.NewRoot
// to test a terminal root panic.
func NewRoot(t testing.TB) *supervise.Group {
	var mu sync.Mutex
	closed := false
	var root *supervise.Group
	root = supervise.NewRoot(func(value any, stack []byte) {
		mu.Lock()
		defer mu.Unlock()
		if !closed {
			t.Errorf("supervised service failed: %v\n%s", value, stack)
			root.Stop()
		}
	})
	t.Cleanup(func() {
		select {
		case <-root.Stop():
		case <-time.After(5 * time.Second):
			t.Errorf("service tree did not stop in 5 s")
		}
		mu.Lock()
		closed = true
		mu.Unlock()
	})
	return root
}

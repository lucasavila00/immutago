// Package supervisetest gives service tests a root group with automatic cleanup.
// NewRoot registers a test cleanup that requests shutdown and waits for all
// children. Tests can use the returned group with supervise.Start and StartGroup.
package supervisetest

import (
	"testing"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// NewRoot creates a root group with reporting disabled and registers its cleanup
// with t. Cleanup waits up to 5 seconds for the group to stop, then reports a test
// failure if any supervised work remains. Run methods must observe cancellation.
func NewRoot(t testing.TB) *supervise.Group {
	root := supervise.NewRoot(nil)
	t.Cleanup(func() {
		select {
		case <-root.Stop():
		case <-time.After(5 * time.Second):
			t.Errorf("service tree did not stop in 5 s")
		}
	})
	return root
}

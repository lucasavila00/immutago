package proctest

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

var release chan struct{}

// stuck is a process that does not stop when its context ends.
type stuck struct {
	im.Process
}

// Run waits for release and ignores its context.
func (*stuck) Run(context.Context) { <-release }

// recorder records the cleanups and the errors of a test.
type recorder struct {
	testing.TB
	cleanups []func()
	errors   int
}

// Cleanup records a cleanup function.
func (r *recorder) Cleanup(cleanup func()) { r.cleanups = append(r.cleanups, cleanup) }

// Errorf counts an error.
func (r *recorder) Errorf(string, ...any) { r.errors++ }

// TestRoot verifies that Root fails a test whose tree does not stop.
func TestRoot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		passing := &recorder{TB: t}
		Root(passing)
		passing.cleanups[0]()
		release = make(chan struct{})
		failing := &recorder{TB: t}
		root := Root(failing)
		proc.Start[stuck](root)
		synctest.Wait()
		failing.cleanups[0]()
		if passing.errors != 0 || failing.errors != 1 {
			t.Fatalf("errors = %d and %d, want 0 and 1", passing.errors, failing.errors)
		}
		close(release)
		<-root.Stop()
	})
}

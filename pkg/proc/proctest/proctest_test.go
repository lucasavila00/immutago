package proctest

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// releaseSignal tells the stuck process to return.
type releaseSignal struct{ im.Scoped }

var release im.Owner[releaseSignal]

// stuck is a process that does not stop when its context ends.
type stuck struct {
	proc.Process[proc.None]
}

// Run waits for release and ignores its context.
func (*stuck) Run(context.Context) {
	for {
		signal := release.Load()
		if signal != nil {
			return
		}
		time.Sleep(time.Second)
	}
}

// releaseStuck lets the stuck process return.
func releaseStuck() { release.Store(&releaseSignal{}) }

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

// TestNewRoot verifies that NewRoot fails a test whose tree does not stop.
func TestNewRoot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		passing := &recorder{TB: t}
		NewRoot(passing)
		passing.cleanups[0]()
		failing := &recorder{TB: t}
		root := NewRoot(failing)
		proc.Start[stuck](root, proc.None{})
		synctest.Wait()
		failing.cleanups[0]()
		if passing.errors != 0 || failing.errors != 1 {
			t.Fatalf("errors = %d and %d, want 0 and 1", passing.errors, failing.errors)
		}
		releaseStuck()
		<-root.Stop()
	})
}

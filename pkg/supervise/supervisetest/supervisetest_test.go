package supervisetest

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

var release chan struct{}

// stuck is a service that does not stop when its context ends.
type stuck struct {
	supervise.Service[supervise.NoArgs]
}

// Run waits for release and ignores its context.
func (*stuck) Run(context.Context) { <-release }

// recorder records the cleanups and the errors of a test.
type recorder struct {
	testing.TB
	mu       sync.Mutex
	cleanups []func()
	errors   int
	messages []string
}

// Cleanup records a cleanup function.
func (r *recorder) Cleanup(cleanup func()) { r.cleanups = append(r.cleanups, cleanup) }

// Errorf counts an error.
func (r *recorder) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors++
	r.messages = append(r.messages, fmt.Sprintf(format, args...))
}

// TestNewRoot verifies that NewRoot fails a test whose tree does not stop.
func TestNewRoot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		passing := &recorder{TB: t}
		NewRoot(passing)
		passing.cleanups[0]()
		release = make(chan struct{})
		failing := &recorder{TB: t}
		root := NewRoot(failing)
		if _, err := supervise.Start[stuck](root, supervise.NoArgs{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		failing.cleanups[0]()
		if passing.errors != 0 || failing.errors != 1 {
			t.Fatalf("errors = %d and %d, want 0 and 1", passing.errors, failing.errors)
		}
		close(release)
		<-root.Stop()
	})
}

// failing records setup and fails in Run.
type failing struct {
	supervise.Service[*atomic.Int32]
}

// Init records the one setup call.
func (s *failing) Init() error {
	s.Args().Add(1)
	return nil
}

// Run supplies a failure that must fail the owning test.
func (*failing) Run(context.Context) { panic("Run failed") }

// TestReportsFailTest uses Stop to prevent the terminal root panic.
func TestReportsFailTest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var inits atomic.Int32
		recorder := &recorder{TB: t}
		root := NewRoot(recorder)
		if _, err := supervise.Start[failing](root, &inits); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		select {
		case <-root.Stop():
		default:
			t.Fatal("failed test root did not stop")
		}
		recorder.cleanups[0]()
		if recorder.errors != 1 || inits.Load() != 1 {
			t.Fatalf("test failures = %d, Init calls = %d; want 1 each", recorder.errors, inits.Load())
		}
		if message := recorder.messages[0]; !strings.Contains(message, "Run failed") ||
			!strings.Contains(message, "failing).Run") {
			t.Errorf("test failure lost its value or frame: %s", message)
		}
	})
}

// lateFailure waits until test cleanup has already reported its timeout.
type lateFailure struct {
	supervise.Service[chan struct{}]
}

// Run fails after the test has stopped accepting reports.
func (s *lateFailure) Run(context.Context) { <-s.Args(); panic("late panic") }

// TestConcurrentFailures reports both attempts while root shutdown starts once.
func TestConcurrentFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		recorder := &recorder{TB: t}
		root := NewRoot(recorder)
		release := make(chan struct{})
		for range 2 {
			if _, err := supervise.Start[lateFailure](root, release); err != nil {
				t.Fatal(err)
			}
		}
		synctest.Wait()
		close(release)
		synctest.Wait()
		select {
		case <-root.Stop():
		default:
			t.Fatal("concurrent failures did not stop the root")
		}
		recorder.cleanups[0]()
		if recorder.errors != 2 {
			t.Fatalf("test failures = %d; want 2", recorder.errors)
		}
	})
}

// TestReportsAfterCleanup ignores failures after a cleanup timeout.
func TestReportsAfterCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		recorder := &recorder{TB: t}
		root := NewRoot(recorder)
		release := make(chan struct{})
		if _, err := supervise.Start[lateFailure](root, release); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		recorder.cleanups[0]()
		close(release)
		<-root.Stop()
		if recorder.errors != 1 || !strings.Contains(recorder.messages[0], "did not stop") {
			t.Fatalf("reports after cleanup changed the test: %v", recorder.messages)
		}
	})
}

package supervise

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

// TestIssue249Probe holds a second late failure until the root has already exited.
func TestIssue249Probe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		reports := make(chan any, 2)
		exited := make(chan struct{})
		top := root(func(any) { close(exited) }, func(value any, _ []byte) {
			reports <- value.(*Failure).Value
		})
		drained, releaseSecond := make(chan struct{}), make(chan struct{})
		_, err := Start[initializing](top, &initHooks{
			init: func(*initializing) error { return nil },
			run:  func(_ *initializing, ctx context.Context) { <-ctx.Done(); <-drained },
		})
		if err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		top.Stop()
		_, err = Start[initializing](top, &initHooks{
			init: func(*initializing) error { return nil },
			run:  func(*initializing, context.Context) { panic("first") },
		})
		if err != nil {
			t.Fatal(err)
		}
		if got := <-reports; got != "first" {
			t.Fatalf("first report = %v", got)
		}
		finished := make(chan struct{})
		go func() {
			defer close(finished)
			_, _ = Start[initializing](top, &initHooks{
				init: func(*initializing) error { return nil },
				run: func(*initializing, context.Context) {
					defer func() { close(drained); <-releaseSecond }()
					panic("second")
				},
			})
		}()
		<-exited
		if len(reports) == 0 {
			t.Error("terminal action ran before the second late failure was reported")
		}
		close(releaseSecond)
		<-finished
		if got := <-reports; got != "second" {
			t.Fatalf("second report = %v", got)
		}
	})
}

// TestIssue250Probe checks that a closed group does not create another service.
func TestIssue250Probe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		exiting, release := make(chan struct{}), make(chan struct{})
		top := root(func(any) { close(exiting); <-release }, func(any, []byte) {})
		group := StartGroup(top)
		<-group.Stop()
		var inits, runs atomic.Int32
		ref, err := Start[initializing](group, &initHooks{
			init: func(*initializing) error { inits.Add(1); return nil },
			run:  func(*initializing, context.Context) { runs.Add(1); panic("late") },
		})
		if err == nil || !ref.IsZero() || inits.Load() != 0 || runs.Load() != 0 {
			t.Errorf("stopped Start: error=%v, zero=%t, Init=%d, Run=%d", err, ref.IsZero(), inits.Load(), runs.Load())
		}
		<-exiting
		close(release)
		<-top.Stop()
	})
}

// TestIssue251Probe holds a parent Init after its child has completed setup.
func TestIssue251Probe(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		exited, fail := make(chan struct{}), make(chan struct{})
		top := root(func(any) { close(exited) }, func(any, []byte) {})
		_, err := Start[initializing](top, &initHooks{
			init: func(*initializing) error { return nil },
			run:  func(*initializing, context.Context) { <-fail; panic("peer") },
		})
		if err != nil {
			t.Fatal(err)
		}
		childReady, releaseInit := make(chan struct{}), make(chan struct{})
		finished := make(chan error, 1)
		var childRuns atomic.Int32
		go func() {
			_, err := Start[initializing](top, &initHooks{
				init: func(p *initializing) error {
					_, err := Start[initializing](p.UnsafeChildren(), &initHooks{
						init: func(*initializing) error { return nil },
						run:  func(*initializing, context.Context) { childRuns.Add(1) },
					})
					close(childReady)
					<-releaseInit
					return err
				},
			})
			finished <- err
		}()
		<-childReady
		close(fail)
		<-exited
		if childRuns.Load() == 0 {
			t.Error("terminal action ran before an initialized child's cleanup Run")
		}
		close(releaseInit)
		if err := <-finished; err != nil {
			t.Fatal(err)
		}
		if childRuns.Load() != 1 {
			t.Fatalf("child Run count = %d", childRuns.Load())
		}
	})
}

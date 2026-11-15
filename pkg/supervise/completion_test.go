package supervise_test

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type completedArgs struct {
	runs          *atomic.Int32
	childStarted  chan struct{}
	childStopping chan struct{}
	releaseChild  chan struct{}
}

type completedService struct {
	supervise.Service[completedArgs]
}

// Run completes once its child has started.
func (s *completedService) Run(context.Context) {
	if s.Args().runs.Add(1) != 1 {
		return
	}
	if _, err := supervise.Start[cleanupChild](s.Children(), s.Args()); err != nil {
		panic(err)
	}
	<-s.Args().childStarted
}

type cleanupChild struct {
	supervise.Service[completedArgs]
}

// Run holds cleanup open until the test releases it.
func (s *cleanupChild) Run(ctx context.Context) {
	close(s.Args().childStarted)
	<-ctx.Done()
	close(s.Args().childStopping)
	<-s.Args().releaseChild
}

type panicSibling struct {
	supervise.Service[supervise.NoArgs]
}

// Run fails each attempt.
func (*panicSibling) Run(context.Context) { panic("restart group") }

// TestCompletedRunRestartsDuringCleanup keeps normal completion through delayed cleanup.
func TestCompletedRunRestartsDuringCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := supervise.NewRoot(nil)
		defer func() { <-root.Stop() }()
		group := supervise.StartGroup(root)
		var runs atomic.Int32
		args := completedArgs{
			runs:          &runs,
			childStarted:  make(chan struct{}),
			childStopping: make(chan struct{}),
			releaseChild:  make(chan struct{}),
		}
		if _, err := supervise.Start[completedService](group, args); err != nil {
			t.Fatal(err)
		}
		<-args.childStopping
		if _, err := supervise.Start[panicSibling](group, supervise.NoArgs{}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(31 * time.Second)
		synctest.Wait()
		close(args.releaseChild)
		synctest.Wait()
		time.Sleep(time.Second)
		synctest.Wait()
		if got := runs.Load(); got != 1 {
			t.Fatalf("Run calls = %d; want 1 after normal return", got)
		}
	})
}

// TestCompletedRunCleanupControl completes cleanup before a sibling fails.
func TestCompletedRunCleanupControl(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := supervise.NewRoot(nil)
		defer func() { <-root.Stop() }()
		group := supervise.StartGroup(root)
		var runs atomic.Int32
		args := completedArgs{
			runs:          &runs,
			childStarted:  make(chan struct{}),
			childStopping: make(chan struct{}),
			releaseChild:  make(chan struct{}),
		}
		if _, err := supervise.Start[completedService](group, args); err != nil {
			t.Fatal(err)
		}
		<-args.childStopping
		close(args.releaseChild)
		synctest.Wait()
		if _, err := supervise.Start[panicSibling](group, supervise.NoArgs{}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(32 * time.Second)
		synctest.Wait()
		if got := runs.Load(); got != 1 {
			t.Fatalf("Run calls = %d; want 1 after normal return", got)
		}
	})
}

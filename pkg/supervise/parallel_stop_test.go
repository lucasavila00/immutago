package supervise

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

type cancelTimes struct {
	at chan<- time.Duration
}

type recordsCancel struct {
	Service[cancelTimes]
}

// Run sends how long after the test start its context ended.
func (s *recordsCancel) Run(ctx context.Context) {
	start := time.Now()
	<-ctx.Done()
	s.Args().at <- time.Since(start)
}

type stuck struct {
	Service[<-chan struct{}]
}

// Run ignores cancellation until the test releases it.
func (s *stuck) Run(context.Context) { <-s.Args() }

// TestParallelStopCancelsSiblingsAtOnce stops a fast service registered before a slow one.
func TestParallelStopCancelsSiblingsAtOnce(t *testing.T) {
	t.Run("parallel", func(t *testing.T) { checkSiblingCancel(t, StartParallelSupervisor, 0) })
	t.Run("ordered", func(t *testing.T) { checkSiblingCancel(t, StartSupervisor, time.Minute) })
}

// checkSiblingCancel stops a group of a fast and a slow service and checks when the fast one is cancelled.
func checkSiblingCancel(t *testing.T, start func(*Supervisor) *Supervisor, want time.Duration) {
	synctest.Test(t, func(t *testing.T) {
		group := start(NewRoot(nil))
		at := make(chan time.Duration, 1)
		if _, err := Start[recordsCancel](group, cancelTimes{at: at}); err != nil {
			t.Fatal(err)
		}
		if _, err := Start[slowToStop](group, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		begin := time.Now()
		<-group.Stop()
		if got := <-at; got != want {
			t.Errorf("first service cancelled after %v; want %v", got, want)
		}
		if got := time.Since(begin); got != time.Minute {
			t.Errorf("stop took %v; want %v", got, time.Minute)
		}
	})
}

// TestParallelStopPastStuckService cancels siblings of a service that does not return.
func TestParallelStopPastStuckService(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		group := StartParallelSupervisor(root)
		at := make(chan time.Duration, 2)
		release := make(chan struct{})
		if _, err := Start[recordsCancel](group, cancelTimes{at: at}); err != nil {
			t.Fatal(err)
		}
		if _, err := Start[stuck](group, (<-chan struct{})(release)); err != nil {
			t.Fatal(err)
		}
		nested := StartSupervisor(group)
		if _, err := Start[recordsCancel](nested, cancelTimes{at: at}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err := group.Shutdown(ctx)
		var shutdown *ShutdownError
		if !errors.As(err, &shutdown) || shutdown.Waiting != "supervise.stuck" {
			t.Fatalf("Shutdown = %v; want a wait for supervise.stuck", err)
		}
		for range 2 {
			if got := <-at; got != 0 {
				t.Errorf("sibling cancelled after %v; want 0", got)
			}
		}
		close(release)
		<-group.Stop()
	})
}

// TestParallelStopOfPendingChildren stops children whose parent Run was never admitted.
func TestParallelStopOfPendingChildren(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		at := make(chan time.Duration, 2)
		owner := newSupervisor(root, false)
		group := StartParallelSupervisor(owner)
		for range 2 {
			if _, err := Start[recordsCancel](group, cancelTimes{at: at}); err != nil {
				t.Fatal(err)
			}
		}
		<-owner.Stop()
		for range 2 {
			if got := <-at; got != 0 {
				t.Errorf("pending child cancelled after %v; want 0", got)
			}
		}
	})
}

// TestParallelSupervisorFromStoppedParent returns a stopped supervisor.
func TestParallelSupervisorFromStoppedParent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		<-root.Stop()
		group := StartParallelSupervisor(root)
		select {
		case <-group.Stop():
		default:
			t.Fatal("supervisor from a stopped parent is not stopped")
		}
		if _, err := Start[recordsCancel](group, cancelTimes{}); !errors.Is(err, context.Canceled) {
			t.Fatalf("Start = %v; want context.Canceled", err)
		}
	})
}

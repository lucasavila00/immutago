package proc

import (
	"context"
	"testing"
	"testing/synctest"
	"time"
)

// TestStopWaitsForParentEntry keeps both Stop channels open until the child watcher finishes.
func TestStopWaitsForParentEntry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := NewRoot(nil)
		nested := &Supervisor{parent: parent, stopped: make(chan struct{})}
		returned, finish := make(chan struct{}), make(chan struct{})
		nested.entry = parent.add(func(ctx context.Context) {
			nested.serve(ctx)
			close(returned)
			<-finish
		})
		synctest.Wait()
		stopped := nested.Stop()
		<-returned
		select {
		case <-stopped:
			t.Error("nested Stop completed before its parent entry finished")
		default:
		}
		parentStopped := parent.Stop()
		synctest.Wait()
		select {
		case <-parentStopped:
			t.Error("parent Stop skipped a child whose watcher is still running")
		default:
		}
		close(finish)
		<-stopped
		<-parentStopped
		if nested.parent != nil || nested.entry != nil {
			t.Error("stopped supervisor keeps its parent links")
		}
	})
}

// TestStopDuringRestartDelay releases a nested supervisor without waiting for the next restart.
func TestStopDuringRestartDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := root(func(value any) { t.Errorf("root escalated: %v", value) }, nil)
		nested := StartSupervisor(parent)
		failures.Store(0)
		Start[failing](nested, None{})
		time.Sleep(31 * time.Second)
		synctest.Wait()
		before := time.Now()
		<-nested.Stop()
		if delay := time.Since(before); delay != 0 {
			t.Errorf("Stop waited for a restart timer: %v", delay)
		}
		parent.mu.Lock()
		remaining := len(parent.children)
		parent.mu.Unlock()
		if remaining != 0 {
			t.Errorf("parent retains %d children after nested Stop", remaining)
		}
		<-parent.Stop()
		takeEvents()
	})
}

// TestStopBeforeParentStart removes an entry that the parent has not launched yet.
func TestStopBeforeParentStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := &Supervisor{stopped: make(chan struct{})}
		nested := StartSupervisor(parent)
		<-nested.Stop()
		if len(parent.children) != 0 || nested.parent != nil || nested.entry != nil {
			t.Error("unstarted supervisor was not released")
		}
		<-parent.Stop()
	})
}

// TestNestedSelfStop checks that a process can stop its own nested supervisor without a wait cycle.
func TestNestedSelfStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := NewRoot(nil)
		selfRoot = StartSupervisor(parent)
		Start[stopper](selfRoot, None{})
		synctest.Wait()
		expectEvents(t, "stopped")
		<-selfRoot.Stop()
		<-parent.Stop()
		takeEvents()
	})
}

// TestStartAfterStopDoesNotRetainChildren checks late process starts and an unrecorded nested entry.
func TestStartAfterStopDoesNotRetainChildren(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := NewRoot(nil)
		<-parent.Stop()
		for range 1000 {
			if Start[first](parent, None{}).Proc() == nil {
				t.Fatal("Start returned an empty PID")
			}
		}
		nested := StartSupervisor(parent)
		parent.mu.Lock()
		remaining := len(parent.children)
		parent.mu.Unlock()
		if remaining != 0 {
			t.Errorf("stopped supervisor retains %d new children", remaining)
		}
		<-nested.Stop()
		if nested.parent != nil || nested.entry != nil {
			t.Error("late nested Stop keeps its parent links")
		}
	})
}

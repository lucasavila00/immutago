package supervise

import (
	"context"
	"testing"
	"testing/synctest"
)

// TestStopWaitsForParentEntry keeps a nested entry until its child cleanup ends.
func TestStopWaitsForParentEntry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := NewRoot(nil)
		nested := StartSupervisor(parent)
		entered, release := make(chan struct{}), make(chan struct{})
		_, err := Start[initializing](nested, &initHooks{
			init: func(*initializing) error { return nil },
			run:  func(_ *initializing, ctx context.Context) { <-ctx.Done(); close(entered); <-release },
		})
		if err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		stopped := nested.Stop()
		<-entered
		parentStopped := parent.Stop()
		synctest.Wait()
		select {
		case <-stopped:
			t.Error("nested Stop completed before cleanup")
		default:
		}
		select {
		case <-parentStopped:
			t.Error("parent Stop completed before nested cleanup")
		default:
		}
		close(release)
		<-stopped
		<-parentStopped
		if nested.parent != nil || nested.entry != nil {
			t.Error("stopped group keeps parent links")
		}
	})
}

// TestStopBeforeParentStart removes an entry that the parent has not launched yet.
func TestStopBeforeParentStart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := newSupervisor(nil, false)
		parent.root = parent
		nested := StartSupervisor(parent)
		<-nested.Stop()
		if parent.children.count != 0 || nested.parent != nil || nested.entry != nil {
			t.Error("unstarted group was not released")
		}
		<-parent.Stop()
	})
}

// TestNestedSelfStop checks that a service can stop its own nested group without a wait cycle.
func TestNestedSelfStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := NewRoot(nil)
		selfRoot = StartSupervisor(parent)
		if _, err := Start[stopper](selfRoot, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		expectEvents(t, "stopped")
		<-selfRoot.Stop()
		<-parent.Stop()
		takeEvents()
	})
}

// TestStartAfterStopDoesNotRetainChildren checks rejected starts and a stopped nested group.
func TestStartAfterStopDoesNotRetainChildren(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := NewRoot(nil)
		<-parent.Stop()
		for range 1000 {
			ref, err := Start[first](parent, NoArgs{})
			if err != context.Canceled || !ref.IsZero() {
				t.Fatalf("stopped Start = %v, %v", ref, err)
			}
		}
		nested := StartSupervisor(parent)
		parent.mu.Lock()
		remaining := parent.children.count
		parent.mu.Unlock()
		if remaining != 0 {
			t.Errorf("stopped group retains %d new children", remaining)
		}
		<-nested.Stop()
		if nested.parent != nil || nested.entry != nil {
			t.Error("late nested Stop keeps its parent links")
		}
		takeEvents()
	})
}

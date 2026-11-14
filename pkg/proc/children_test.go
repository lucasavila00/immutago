package proc

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
)

// pool is a process that starts three children under its own supervisor.
type pool struct{ Process[None] }

// Run starts the children, one of them in a nested supervisor, and waits until ctx ends.
func (p *pool) Run(ctx context.Context) {
	record("start pool")
	sup := p.Supervisor()
	Start[first](sup, None{})
	Start[second](StartSupervisor(sup), None{})
	Start[third](sup, None{})
	<-ctx.Done()
	record("stop pool")
}

var (
	poolRuns    atomic.Int32
	poolRelease chan struct{}
)

// restartedPool is a process that starts two children and panics on its first run.
type restartedPool struct{ Process[None] }

// Run starts the children, panics on the first run when released, and otherwise waits until ctx ends.
func (p *restartedPool) Run(ctx context.Context) {
	Start[first](p.Supervisor(), None{})
	Start[second](p.Supervisor(), None{})
	if poolRuns.Add(1) == 1 {
		<-poolRelease
		panic("pool")
	}
	<-ctx.Done()
}

// failingParent is a process with a child that panics on each run.
type failingParent struct{ Process[None] }

// Run starts the failing child and waits until ctx ends.
func (p *failingParent) Run(ctx context.Context) {
	record("start parent")
	Start[failing](p.Supervisor(), None{})
	<-ctx.Done()
	record("stop parent")
}

// TestChildrenStopBeforeRun verifies that a stopped process stops its children last started first
// before the context of its run ends.
func TestChildrenStopBeforeRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot()
		Start[pool](root, None{})
		synctest.Wait()
		expectStarted(t, "start pool", "start first", "start second", "start third")
		<-root.Stop()
		expectEvents(t, "stop third", "stop second", "stop first", "stop pool")
	})
}

// askerArgs is the argument of an asker.
type askerArgs struct {
	im.Shared
	Self PID[asker]
}

// asker is a process whose Run answers the calls of its child.
type asker struct {
	Process[askerArgs]
	requests chan chan int
}

// Run starts a child that calls Ask and answers each request until ctx ends.
func (a *asker) Run(ctx context.Context) {
	a.requests = make(chan chan int)
	Start[caller](a.Supervisor(), a.Args().Self)
	for {
		select {
		case <-ctx.Done():
			record("stop asker")
			return
		case reply := <-a.requests:
			reply <- 1
		}
	}
}

// Ask sends a request to Run and waits for the answer.
func (a *asker) Ask() int {
	reply := make(chan int)
	a.requests <- reply
	return <-reply
}

// caller is a process that calls its parent until ctx ends.
type caller struct{ Process[PID[asker]] }

// Run calls Ask on the parent until ctx ends.
func (c *caller) Run(ctx context.Context) {
	for ctx.Err() == nil {
		c.Args().Proc().Ask()
		time.Sleep(time.Millisecond)
	}
	record("stop caller")
}

// TestChildCallsParentDuringStop verifies that a child can call its parent until the child stops.
func TestChildCallsParentDuringStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot()
		pid := Reserve[asker]()
		StartReserved(root, pid, askerArgs{Self: pid})
		time.Sleep(time.Second)
		<-root.Stop()
		expectEvents(t, "stop caller", "stop asker")
	})
}

// TestChildrenRestartWithRun verifies that a restart stops the children of the old run and the new run starts them.
func TestChildrenRestartWithRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		poolRuns.Store(0)
		poolRelease = make(chan struct{})
		root := root(func(value any) { t.Errorf("escalated %v", value) })
		Start[restartedPool](root, None{})
		synctest.Wait()
		expectStarted(t, "start first", "start second")
		close(poolRelease)
		synctest.Wait()
		expectEvents(t, "stop second", "stop first")
		time.Sleep(time.Second)
		synctest.Wait()
		expectStarted(t, "start first", "start second")
		<-root.Stop()
		expectEvents(t, "stop second", "stop first")
	})
}

// TestChildEscalationFailsRun verifies that a child past its restart limit fails the run of its parent.
func TestChildEscalationFailsRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		failures.Store(0)
		root := root(func(value any) { t.Errorf("escalated %v", value) })
		Start[failingParent](root, None{})
		time.Sleep(31 * time.Second)
		synctest.Wait()
		expectEvents(t, "start parent", "00:00:00", "00:00:01", "00:00:03", "00:00:07", "00:00:15", "00:00:31",
			"stop parent")
		time.Sleep(time.Second)
		synctest.Wait()
		expectEvents(t, "start parent", "00:00:32")
		<-root.Stop()
		expectEvents(t, "stop parent")
	})
}

// variantArgs is the argument of a variant.
type variantArgs struct {
	im.Shared
	Name string
}

// variant is a process that a router starts for each name.
type variant struct{ Process[variantArgs] }

// Run records the start and the stop.
func (v *variant) Run(ctx context.Context) { track(ctx, v.Args().Name) }

// router is a process whose methods add and remove variants during one run.
type router struct {
	Process[None]
	mu       sync.Mutex
	variants map[string]*Supervisor
}

// Run waits until ctx ends.
func (*router) Run(ctx context.Context) { <-ctx.Done() }

// Add starts a variant under its own nested supervisor in the supervisor of the run.
func (r *router) Add(name string) {
	sub := StartSupervisor(r.Supervisor())
	Start[variant](sub, variantArgs{Name: name})
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.variants == nil {
		r.variants = make(map[string]*Supervisor)
	}
	r.variants[name] = sub
}

// Remove stops one variant and waits for it.
func (r *router) Remove(name string) {
	r.mu.Lock()
	sub := r.variants[name]
	delete(r.variants, name)
	r.mu.Unlock()
	<-sub.Stop()
}

// entries returns the number of children in the supervisor of the run.
func (r *router) entries() int {
	s := r.Supervisor()
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.children)
}

// TestChildrenChangeDuringRun verifies that methods can start and stop single children many times in one run.
// synctest fails the test if a goroutine of the tree remains after the root stops.
func TestChildrenChangeDuringRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot()
		pid := Start[router](root, None{})
		pid.Proc().Add("a")
		synctest.Wait()
		expectEvents(t, "start a")
		for range 100 {
			pid.Proc().Add("old")
			synctest.Wait()
			pid.Proc().Remove("old")
			expectEvents(t, "start old", "stop old")
		}
		if entries := pid.Proc().entries(); entries != 1 {
			t.Fatalf("supervisor of the run has %d entries after the removals, want 1", entries)
		}
		pid.Proc().Add("b")
		pid.Proc().Add("c")
		synctest.Wait()
		expectStarted(t, "start b", "start c")
		pid.Proc().Remove("b")
		expectEvents(t, "stop b")
		<-root.Stop()
		expectEvents(t, "stop c", "stop a")
		pid.Proc().Add("late")
		synctest.Wait()
		expectEvents(t)
	})
}

// TestChildrenOfProcessThatDoesNotRun verifies that children of a process that never runs do not run.
func TestChildrenOfProcessThatDoesNotRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot()
		<-root.Stop()
		pid := Start[router](root, None{})
		pid.Proc().Add("a")
		synctest.Wait()
		expectEvents(t)
	})
}

// returning is a process whose Run returns at once.
type returning struct{ Process[None] }

// Run returns at once.
func (*returning) Run(context.Context) {}

// TestReturnedChildrenAreRemoved verifies that a supervisor removes children whose Run returned.
func TestReturnedChildrenAreRemoved(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot()
		Start[first](root, None{})
		for range 1000 {
			Start[returning](root, None{})
		}
		synctest.Wait()
		root.mu.Lock()
		entries := len(root.children)
		root.mu.Unlock()
		if entries != 1 {
			t.Fatalf("root has %d entries, want 1", entries)
		}
		takeEvents()
		<-root.Stop()
		expectEvents(t, "stop first")
	})
}

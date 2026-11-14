package proc

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
)

// TestGoexitStopsWatcher checks completion when a child ends its goroutine.
func TestGoexitStopsWatcher(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		root.add(func(context.Context) { runtime.Goexit() })
		synctest.Wait()
		done := root.Stop()
		synctest.Wait()
		select {
		case <-done:
		default:
			t.Fatal("supervisor did not stop after Goexit")
		}
	})
}

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
		root := NewRoot(nil)
		Start[pool](root, None{})
		synctest.Wait()
		expectStarted(t, "start pool", "start first", "start second", "start third")
		<-root.Stop()
		expectEvents(t, "stop third", "stop second", "stop first", "stop pool")
	})
}

// TestSupervisorDuringHalt checks that a late child cannot start while the run stops.
func TestSupervisorDuringHalt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		entered := make(chan struct{})
		resume := make(chan struct{})
		var once sync.Once
		p := &Process[None]{
			cancel:  func() { once.Do(func() { close(entered); <-resume }) },
			unwatch: func() bool { return true },
		}
		halted := make(chan struct{})
		go func() { p.halt(); close(halted) }()
		<-entered
		s := p.Supervisor()
		ran := make(chan struct{})
		s.add(func(context.Context) { close(ran) })
		synctest.Wait()
		close(resume)
		<-halted
		p.end()
		select {
		case <-ran:
			t.Fatal("child started after halt began")
		default:
		}
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
		root := NewRoot(nil)
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
		root := root(func(value any) { t.Errorf("escalated %v", value) }, nil)
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
		root := root(func(value any) { t.Errorf("escalated %v", value) }, nil)
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
		root := NewRoot(nil)
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
		root := NewRoot(nil)
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
		root := NewRoot(nil)
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

// shortParent is a process whose Run starts two children and returns after 1 s.
type shortParent struct{ Process[None] }

// Run starts the children and returns after 1 s.
func (p *shortParent) Run(context.Context) {
	Start[first](p.Supervisor(), None{})
	Start[second](p.Supervisor(), None{})
	time.Sleep(time.Second)
}

// TestChildrenStopWhenRunReturns verifies that the children stop when Run returns and the process ends.
func TestChildrenStopWhenRunReturns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := root(func(value any) { t.Errorf("escalated %v", value) }, nil)
		Start[shortParent](root, None{})
		synctest.Wait()
		expectStarted(t, "start first", "start second")
		time.Sleep(time.Minute)
		synctest.Wait()
		expectEvents(t, "stop second", "stop first")
		<-root.Stop()
		expectEvents(t)
	})
}

// slowStop is a process that takes 10 s to stop.
type slowStop struct{ Process[None] }

// Run waits until ctx ends, then waits 10 s.
func (*slowStop) Run(ctx context.Context) {
	<-ctx.Done()
	time.Sleep(10 * time.Second)
	record("stop slow")
}

// stoppingParent is a process with a failing child and a slow child.
type stoppingParent struct{ Process[None] }

// Run starts the failing child, then the slow child, and waits until ctx ends.
func (p *stoppingParent) Run(ctx context.Context) {
	Start[failing](p.Supervisor(), None{})
	Start[slowStop](p.Supervisor(), None{})
	<-ctx.Done()
	record("stop parent")
}

// TestChildEscalatesDuringStop verifies that an escalation during Stop does not block or restart the parent.
func TestChildEscalatesDuringStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		failures.Store(0)
		root := root(func(value any) { t.Errorf("escalated %v", value) }, nil)
		Start[stoppingParent](root, None{})
		time.Sleep(30 * time.Second)
		synctest.Wait()
		takeEvents()
		<-root.Stop()
		expectEvents(t, "00:00:31", "stop slow", "stop parent")
	})
}

// lateParent is a process that tries to start a child after its child escalated.
type lateParent struct{ Process[None] }

// Run starts a failing child, waits until ctx ends, and then starts another child.
func (p *lateParent) Run(ctx context.Context) {
	Start[failing](p.Supervisor(), None{})
	<-ctx.Done()
	Start[first](p.Supervisor(), None{})
	record("late start")
}

// TestStartAfterEscalation verifies that a child started after an escalation in the same run does not run.
func TestStartAfterEscalation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		failures.Store(0)
		root := root(func(value any) { t.Errorf("escalated %v", value) }, nil)
		Start[lateParent](root, None{})
		time.Sleep(31 * time.Second)
		synctest.Wait()
		expectEvents(t, "00:00:00", "00:00:01", "00:00:03", "00:00:07", "00:00:15", "00:00:31", "late start")
		<-root.Stop()
		takeEvents()
	})
}

// TestMethodsRaceStop verifies that children started by methods during Stop either stop or never run.
func TestMethodsRaceStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		pid := Start[router](root, None{})
		synctest.Wait()
		var callers sync.WaitGroup
		callers.Go(func() {
			for range 200 {
				pid.Proc().Add("x")
			}
		})
		<-root.Stop()
		callers.Wait()
		synctest.Wait()
		starts, stops := 0, 0
		for _, event := range takeEvents() {
			switch event {
			case "start x":
				starts++
			case "stop x":
				stops++
			}
		}
		if starts != stops {
			t.Fatalf("%d starts and %d stops, want the same count", starts, stops)
		}
	})
}

// namedVariant is a process that panics on each run if its name is bad.
type namedVariant struct{ Process[variantArgs] }

// Run panics for the bad variant and otherwise records the start and the stop.
func (v *namedVariant) Run(ctx context.Context) {
	if v.Args().Name == "bad" {
		panic("bad variant")
	}
	track(ctx, v.Args().Name)
}

// variantNames is the argument of a rebuilding dispatcher.
type variantNames struct {
	im.Shared
	Names im.FrozenSlice[string]
}

// rebuilding is a dispatcher that starts one group for each variant in Run.
type rebuilding struct{ Process[variantNames] }

// Run starts one nested group for each variant and waits until ctx ends.
func (r *rebuilding) Run(ctx context.Context) {
	for _, name := range r.Args().Names.All() {
		Start[namedVariant](StartSupervisor(r.Supervisor()), variantArgs{Name: name})
	}
	<-ctx.Done()
}

// TestGroupEscalation verifies that a failing group restarts alone at first, then fails the run, and the new run
// starts the groups again.
func TestGroupEscalation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := root(func(value any) { t.Errorf("escalated %v", value) }, nil)
		Start[rebuilding](root, variantNames{Names: im.FreezeSlice([]string{"good", "bad"})})
		time.Sleep(216 * time.Second)
		synctest.Wait()
		expectEvents(t, "start good")
		time.Sleep(2 * time.Second)
		synctest.Wait()
		expectEvents(t, "stop good", "start good")
		<-root.Stop()
		expectEvents(t, "stop good")
	})
}

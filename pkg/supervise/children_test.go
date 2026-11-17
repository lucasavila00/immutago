package supervise

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
		root.add(func(context.Context) { runtime.Goexit() }, nil)
		synctest.Wait()
		done := root.Stop()
		synctest.Wait()
		select {
		case <-done:
		default:
			t.Fatal("group did not stop after Goexit")
		}
	})
}

// pool is a service that starts three children under its own group.
type pool struct {
	Service[NoArgs]
	supervisor *Supervisor
}

// Init saves the supervisor for later child starts.
func (p *pool) Init(supervisor *Supervisor) error { p.supervisor = supervisor; return nil }

// Run starts the children, one of them in a nested group, and waits until ctx ends.
func (p *pool) Run(ctx context.Context) {
	record("start pool")
	sup := p.supervisor
	if _, err := Start[first](sup, NoArgs{}); err != nil {
		panic(err)
	}
	if _, err := Start[second](StartSupervisor(sup), NoArgs{}); err != nil {
		panic(err)
	}
	if _, err := Start[third](sup, NoArgs{}); err != nil {
		panic(err)
	}
	<-ctx.Done()
	record("stop pool")
}

var (
	poolRuns    atomic.Int32
	poolRelease chan struct{}
)

// failingPool is a service that starts two children and panics when released.
type failingPool struct {
	Service[NoArgs]
	supervisor *Supervisor
}

// Init saves the supervisor for later child starts.
func (p *failingPool) Init(supervisor *Supervisor) error { p.supervisor = supervisor; return nil }

// Run starts the children and panics when released.
func (p *failingPool) Run(context.Context) {
	if _, err := Start[first](p.supervisor, NoArgs{}); err != nil {
		panic(err)
	}
	if _, err := Start[second](p.supervisor, NoArgs{}); err != nil {
		panic(err)
	}
	poolRuns.Add(1)
	<-poolRelease
	panic("pool")
}

// failingParent is a service with a child that panics on each run.
type failingParent struct {
	Service[NoArgs]
	supervisor *Supervisor
}

// Init saves the supervisor for later child starts.
func (p *failingParent) Init(supervisor *Supervisor) error { p.supervisor = supervisor; return nil }

// Run starts the failing child and waits until ctx ends.
func (p *failingParent) Run(ctx context.Context) {
	record("start parent")
	if _, err := Start[failing](p.supervisor, NoArgs{}); err != nil {
		panic(err)
	}
	<-ctx.Done()
	record("stop parent")
}

// TestChildrenStopBeforeRun verifies that a stopped service stops children in reverse registration order
// before the context of its run ends.
func TestChildrenStopBeforeRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		if _, err := Start[pool](root, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		expectStarted(t, "start pool", "start first", "start second", "start third")
		<-root.Stop()
		expectEvents(t, "stop third", "stop second", "stop first", "stop pool")
	})
}

// TestChildrenDuringHalt rejects a child while its owner's children stop.
func TestChildrenDuringHalt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		parent := NewRoot(nil)
		defer func() { <-parent.Stop() }()
		entered, resume := make(chan struct{}), make(chan struct{})
		owner, err := Start[initializing](parent, &initHooks{
			init: func(p *initializing) error {
				_, err := Start[initializing](p.supervisor, &initHooks{
					init: func(*initializing) error { return nil },
					run: func(_ *initializing, ctx context.Context) {
						<-ctx.Done()
						close(entered)
						<-resume
					},
				})
				return err
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		done := parent.Stop()
		<-entered
		var inits, runs int
		ref, err := Start[initializing](owner.Borrow().supervisor, &initHooks{
			init: func(*initializing) error { inits++; return nil },
			run:  func(*initializing, context.Context) { runs++ },
		})
		close(resume)
		<-done
		if err != context.Canceled || !ref.IsZero() || inits != 0 || runs != 0 {
			t.Fatalf("closed child start: err = %v, zero = %v, Init = %d, Run = %d", err, ref.IsZero(), inits, runs)
		}
	})
}

// asker is a service whose Run answers the calls of its child.
type asker struct {
	Service[NoArgs]
	requests   chan chan int
	supervisor *Supervisor
}

// Init creates the request channel before publication.
func (a *asker) Init(supervisor *Supervisor) error {
	a.requests = make(chan chan int)
	a.supervisor = supervisor
	return nil
}

// StartCaller creates a child after its parent's reference is available.
func (a *asker) StartCaller(self Ref[asker]) {
	if _, err := Start[caller](a.supervisor, self); err != nil {
		panic(err)
	}
}

// Run answers requests until its context ends.
func (a *asker) Run(ctx context.Context) {
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

// caller is a service that calls its parent until ctx ends.
type caller struct{ Service[Ref[asker]] }

// Run calls Ask on the parent until ctx ends.
func (c *caller) Run(ctx context.Context) {
	for ctx.Err() == nil {
		c.Args().Borrow().Ask()
		time.Sleep(time.Millisecond)
	}
	record("stop caller")
}

// TestChildCallsParentDuringStop verifies that a child can call its parent until the child stops.
func TestChildCallsParentDuringStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		ref, err := Start[asker](root, NoArgs{})
		if err != nil {
			t.Fatal(err)
		}
		ref.Borrow().StartCaller(ref)
		time.Sleep(time.Second)
		<-root.Stop()
		expectEvents(t, "stop caller", "stop asker")
	})
}

// TestChildrenStopAfterFailure checks reverse child cleanup after the first failure.
func TestChildrenStopAfterFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		poolRuns.Store(0)
		poolRelease = make(chan struct{})
		terminal := make(chan any, 1)
		root := root(func(value any) { terminal <- value }, nil)
		if _, err := Start[failingPool](root, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		expectStarted(t, "start first", "start second")
		close(poolRelease)
		if failure := (<-terminal).(*Failure); failure.Value != "pool" {
			t.Fatalf("failure = %v", failure)
		}
		<-root.Stop()
		expectEvents(t, "stop second", "stop first")
		if poolRuns.Load() != 1 {
			t.Fatal("failed parent ran more than once")
		}
	})
}

// TestChildEscalationFailsRun checks that the first child failure stops the parent.
func TestChildEscalationFailsRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		failures.Store(0)
		root := root(func(any) {}, nil)
		if _, err := Start[failingParent](root, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		expectEvents(t, "start parent", "00:00:00",
			"stop parent")
		<-root.Stop()
		expectEvents(t)
	})
}

// variantArgs is the argument of a variant.
type variantArgs struct {
	im.Frozen
	Name string
}

// variant is a service that a router starts for each name.
type variant struct{ Service[variantArgs] }

// Run records the start and the stop.
func (v *variant) Run(ctx context.Context) { track(ctx, v.Args().Name) }

// router is a service whose methods add and remove variants during one run.
type router struct {
	Service[NoArgs]
	mu         sync.Mutex
	variants   map[string]*Supervisor
	supervisor *Supervisor
}

// Init saves the supervisor for later child starts.
func (r *router) Init(supervisor *Supervisor) error { r.supervisor = supervisor; return nil }

// Run waits until ctx ends.
func (*router) Run(ctx context.Context) { <-ctx.Done() }

// Add starts a variant under its own nested group in the group of the run.
func (r *router) Add(name string) bool {
	sub := StartSupervisor(r.supervisor)
	if _, err := Start[variant](sub, variantArgs{Name: name}); err != nil {
		if err == context.Canceled {
			return false
		}
		panic(err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.variants == nil {
		r.variants = make(map[string]*Supervisor)
	}
	r.variants[name] = sub
	return true
}

// Remove stops one variant and waits for it.
func (r *router) Remove(name string) {
	r.mu.Lock()
	sub := r.variants[name]
	delete(r.variants, name)
	r.mu.Unlock()
	<-sub.Stop()
}

// entries returns the number of children in the group of the run.
func (r *router) entries() int {
	s := r.supervisor
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.children.count
}

// TestChildrenChangeDuringRun verifies that methods can start and stop single children many times in one run.
// synctest fails the test if a goroutine of the tree remains after the root stops.
func TestChildrenChangeDuringRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		ref, _ := Start[router](root, NoArgs{})
		ref.Borrow().Add("a")
		synctest.Wait()
		expectEvents(t, "start a")
		for range 100 {
			ref.Borrow().Add("old")
			synctest.Wait()
			ref.Borrow().Remove("old")
			expectEvents(t, "start old", "stop old")
		}
		if entries := ref.Borrow().entries(); entries != 1 {
			t.Fatalf("group of the run has %d entries after the removals, want 1", entries)
		}
		ref.Borrow().Add("b")
		ref.Borrow().Add("c")
		synctest.Wait()
		expectStarted(t, "start b", "start c")
		ref.Borrow().Remove("b")
		expectEvents(t, "stop b")
		<-root.Stop()
		expectEvents(t, "stop c", "stop a")
		if ref.Borrow().Add("late") {
			t.Error("child Start succeeded after parent Stop")
		}
		synctest.Wait()
		expectEvents(t)
	})
}

// TestChildrenAfterParentStop rejects children after their parent stopped.
func TestChildrenAfterParentStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		ref, _ := Start[router](root, NoArgs{})
		<-root.Stop()
		if ref.Borrow().Add("a") {
			t.Error("child Start succeeded after parent Stop")
		}
		synctest.Wait()
		expectEvents(t)
	})
}

// returning is a service whose Run returns at once.
type returning struct{ Service[NoArgs] }

// Run returns at once.
func (*returning) Run(context.Context) {}

// TestReturnedChildrenAreRemoved verifies that a group removes children whose Run returned.
func TestReturnedChildrenAreRemoved(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		if _, err := Start[first](root, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		for range 1000 {
			if _, err := Start[returning](root, NoArgs{}); err != nil {
				t.Fatal(err)
			}
		}
		synctest.Wait()
		root.mu.Lock()
		entries := root.children.count
		root.mu.Unlock()
		if entries != 1 {
			t.Fatalf("root has %d entries, want 1", entries)
		}
		takeEvents()
		<-root.Stop()
		expectEvents(t, "stop first")
	})
}

// shortParent is a service whose Run starts two children and returns after 1 s.
type shortParent struct {
	Service[NoArgs]
	supervisor *Supervisor
}

// Init saves the supervisor for later child starts.
func (p *shortParent) Init(supervisor *Supervisor) error { p.supervisor = supervisor; return nil }

// Run starts the children and returns after 1 s.
func (p *shortParent) Run(context.Context) {
	if _, err := Start[first](p.supervisor, NoArgs{}); err != nil {
		panic(err)
	}
	if _, err := Start[second](p.supervisor, NoArgs{}); err != nil {
		panic(err)
	}
	time.Sleep(time.Second)
}

// TestChildrenStopWhenRunReturns verifies that the children stop when Run returns and the service ends.
func TestChildrenStopWhenRunReturns(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := root(func(value any) { t.Errorf("escalated %v", value) }, nil)
		if _, err := Start[shortParent](root, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		expectStarted(t, "start first", "start second")
		time.Sleep(time.Minute)
		synctest.Wait()
		expectEvents(t, "stop second", "stop first")
		<-root.Stop()
		expectEvents(t)
	})
}

// slowStop is a service that takes 10 s to stop.
type slowStop struct{ Service[NoArgs] }

// Run waits until ctx ends, then waits 10 s.
func (*slowStop) Run(ctx context.Context) {
	<-ctx.Done()
	time.Sleep(10 * time.Second)
	record("stop slow")
}

// stoppingParent is a service with a failing child and a slow child.
type stoppingParent struct {
	Service[chan struct{}]
	supervisor *Supervisor
}

// Init saves the supervisor for later child starts.
func (p *stoppingParent) Init(supervisor *Supervisor) error { p.supervisor = supervisor; return nil }

// Run starts the failing child, then the slow child, and waits until ctx ends.
func (p *stoppingParent) Run(ctx context.Context) {
	if _, err := Start[initializing](p.supervisor, &initHooks{
		init: func(*initializing) error { return nil },
		run:  func(*initializing, context.Context) { <-p.Args(); panic("during Stop") },
	}); err != nil {
		panic(err)
	}
	if _, err := Start[slowStop](p.supervisor, NoArgs{}); err != nil {
		panic(err)
	}
	<-ctx.Done()
	record("stop parent")
}

// TestChildFailsDuringStop preserves a failure while children stop in order.
func TestChildFailsDuringStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		failures.Store(0)
		terminal := make(chan any, 1)
		root := root(func(value any) { terminal <- value }, nil)
		release := make(chan struct{})
		if _, err := Start[stoppingParent](root, release); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		takeEvents()
		done := root.Stop()
		close(release)
		<-done
		if failure := (<-terminal).(*Failure); failure.Value != "during Stop" {
			t.Fatalf("terminal failure = %v", failure)
		}
		expectEvents(t, "stop slow", "stop parent")
	})
}

// lateParent is a service that tries to start a child after its child escalated.
type lateParent struct {
	Service[NoArgs]
	supervisor *Supervisor
}

// Init saves the supervisor for later child starts.
func (p *lateParent) Init(supervisor *Supervisor) error { p.supervisor = supervisor; return nil }

// Run starts a failing child, waits until ctx ends, and then starts another child.
func (p *lateParent) Run(ctx context.Context) {
	if _, err := Start[failing](p.supervisor, NoArgs{}); err != nil {
		panic(err)
	}
	<-ctx.Done()
	ref, err := Start[first](p.supervisor, NoArgs{})
	if err != context.Canceled || !ref.IsZero() {
		panic("late Start was accepted")
	}
	record("late start rejected")
}

// TestStartAfterEscalation rejects a child before Init.
func TestStartAfterEscalation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		failures.Store(0)
		root := root(func(any) {}, nil)
		if _, err := Start[lateParent](root, NoArgs{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		expectEvents(t, "00:00:00", "late start rejected")
		<-root.Stop()
		takeEvents()
	})
}

// TestMethodsRaceStop gives each initialized child exactly one completed Run.
func TestMethodsRaceStop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		root := NewRoot(nil)
		ref, _ := Start[router](root, NoArgs{})
		synctest.Wait()
		accepted := 0
		if ref.Borrow().Add("x") {
			accepted++
		}
		var callers sync.WaitGroup
		callers.Go(func() {
			for range 200 {
				if ref.Borrow().Add("x") {
					accepted++
				}
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
		if starts != accepted || stops != accepted {
			t.Fatalf("%d starts and %d stops, want %d of each", starts, stops, accepted)
		}
	})
}

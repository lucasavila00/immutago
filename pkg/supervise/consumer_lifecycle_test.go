package supervise

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

// consumerLifetime exposes the start and cleanup of one background run.
type consumerLifetime struct {
	started  chan struct{}
	stopping chan struct{}
	stopped  chan struct{}
	finish   <-chan struct{}
}

// newConsumerLifetime creates signals for one background run.
func newConsumerLifetime() consumerLifetime {
	return consumerLifetime{started: make(chan struct{}), stopping: make(chan struct{}), stopped: make(chan struct{})}
}

// run waits for cancellation and optional cleanup work.
func (l consumerLifetime) run(ctx context.Context) {
	close(l.started)
	<-ctx.Done()
	close(l.stopping)
	if l.finish != nil {
		<-l.finish
	}
	close(l.stopped)
}

// checkRunning checks that cancellation has not started.
func (l consumerLifetime) checkRunning(t *testing.T) {
	t.Helper()
	select {
	case <-l.stopping:
		t.Fatal("unrelated background run was cancelled")
	default:
	}
}

type consumerModelArgs struct {
	version int
	life    consumerLifetime
}

type consumerModel struct{ Service[consumerModelArgs] }

// Run keeps the model active until cancellation.
func (s *consumerModel) Run(ctx context.Context) { s.Args().life.run(ctx) }

// Version reads the model version selected by the caller.
func (s *consumerModel) Version() int { return s.Args().version }

type consumerParentArgs struct {
	starts  *atomic.Int32
	workers chan consumerLifetime
	runs    chan int32
	fail    <-chan struct{}
	finish  <-chan struct{}
}

type consumerParent struct {
	Service[consumerParentArgs]
	generation int32
	supervisor *Supervisor
}

// Init counts each new parent instance.
func (s *consumerParent) Init(supervisor *Supervisor) error {
	s.supervisor = supervisor
	s.generation = s.Args().starts.Add(1)
	return nil
}

// Run starts its worker and waits for the test to release its failure.
func (s *consumerParent) Run(context.Context) {
	args := s.Args()
	worker := newConsumerLifetime()
	worker.finish = args.finish
	if _, err := Start[consumerModel](s.supervisor, consumerModelArgs{life: worker}); err != nil {
		panic(err)
	}
	args.workers <- worker
	args.runs <- s.generation
	<-args.fail
	panic("model worker failed")
}

// Generation reads the initialized parent instance number.
func (s *consumerParent) Generation() int32 { return s.generation }

// TestConsumerParentFailure waits for owned children before stopping an older peer.
func TestConsumerParentFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		terminal := make(chan any, 1)
		group := root(func(value any) { terminal <- value }, nil)
		peer := newConsumerLifetime()
		if _, err := Start[consumerModel](group, consumerModelArgs{version: 99, life: peer}); err != nil {
			t.Fatal(err)
		}
		<-peer.started
		fail, finish := make(chan struct{}), make(chan struct{})
		args := consumerParentArgs{
			starts: new(atomic.Int32), workers: make(chan consumerLifetime, 2),
			runs: make(chan int32, 2), fail: fail, finish: finish,
		}
		ref, err := Start[consumerParent](group, args)
		if err != nil {
			t.Fatal(err)
		}
		oldWorker := <-args.workers
		<-oldWorker.started
		if run := <-args.runs; run != 1 {
			t.Fatalf("first run = %d, want 1", run)
		}
		close(fail)
		<-oldWorker.stopping
		synctest.Wait()
		if ref.Borrow().Generation() != 1 || args.starts.Load() != 1 {
			t.Fatal("failure initialized another parent")
		}
		peer.checkRunning(t)
		close(finish)
		<-oldWorker.stopped
		if failure := (<-terminal).(*Failure); failure.Value != "model worker failed" {
			t.Fatalf("terminal failure = %v", failure)
		}
		<-group.Stop()
		if args.starts.Load() != 1 || len(args.workers) != 0 || len(args.runs) != 0 {
			t.Error("failed parent started another instance or worker")
		}
		<-peer.stopped
	})
}

type consumerRouterArgs struct {
	model Ref[consumerModel]
	life  consumerLifetime
}

type consumerRouter struct {
	Service[consumerRouterArgs]
	mu    sync.RWMutex
	model Ref[consumerModel]
}

// Init selects the initial model before publication.
func (s *consumerRouter) Init(*Supervisor) error {
	s.model = s.Args().model
	return nil
}

// Run keeps the router active until cancellation.
func (s *consumerRouter) Run(ctx context.Context) { s.Args().life.run(ctx) }

// Select publishes a new model reference under the router lock.
func (s *consumerRouter) Select(model Ref[consumerModel]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.model = model
}

// Version calls the model selected under the router lock.
func (s *consumerRouter) Version() int {
	s.mu.RLock()
	model := s.model
	s.mu.RUnlock()
	return model.Borrow().Version()
}

// TestConsumerModelSwap stops selected versions while the router and peers run.
func TestConsumerModelSwap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		group := NewRoot(nil)
		models := StartSupervisor(group)
		oldVersion := StartSupervisor(models)
		oldLife, otherLife, routerLife := newConsumerLifetime(), newConsumerLifetime(), newConsumerLifetime()
		oldModel, err := Start[consumerModel](oldVersion, consumerModelArgs{version: 1, life: oldLife})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Start[consumerModel](group, consumerModelArgs{version: 99, life: otherLife}); err != nil {
			t.Fatal(err)
		}
		router, err := Start[consumerRouter](group, consumerRouterArgs{model: oldModel, life: routerLife})
		if err != nil {
			t.Fatal(err)
		}
		<-oldLife.started
		<-otherLife.started
		<-routerLife.started
		if router.Borrow().Version() != 1 {
			t.Fatal("router did not select the initial model")
		}
		challenger := StartSupervisor(models)
		newLife := newConsumerLifetime()
		newModel, err := Start[consumerModel](challenger, consumerModelArgs{version: 2, life: newLife})
		if err != nil {
			t.Fatal(err)
		}
		<-newLife.started
		router.Borrow().Select(newModel)
		if router.Borrow().Version() != 2 {
			t.Fatal("router did not select the new model before old cleanup")
		}
		<-oldVersion.Stop()
		<-oldLife.stopped
		synctest.Wait()
		newLife.checkRunning(t)
		otherLife.checkRunning(t)
		routerLife.checkRunning(t)
		if router.Borrow().Version() != 2 {
			t.Fatal("old version cleanup changed the selected model")
		}
		<-challenger.Stop()
		<-newLife.stopped
		synctest.Wait()
		otherLife.checkRunning(t)
		routerLife.checkRunning(t)
		<-group.Stop()
		<-otherLife.stopped
		<-routerLife.stopped
	})
}

package supervise

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
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
}

// Init counts each new parent instance.
func (s *consumerParent) Init() error {
	s.generation = s.Args().starts.Add(1)
	return nil
}

// Run starts its worker and fails only on the first parent instance.
func (s *consumerParent) Run(ctx context.Context) {
	args := s.Args()
	worker := newConsumerLifetime()
	if s.generation == 1 {
		worker.finish = args.finish
	}
	if _, err := Start[consumerModel](s.Children(), consumerModelArgs{life: worker}); err != nil {
		panic(err)
	}
	args.workers <- worker
	args.runs <- s.generation
	if s.generation == 1 {
		<-args.fail
		panic("model worker failed")
	}
	<-ctx.Done()
}

// Generation reads the initialized parent instance number.
func (s *consumerParent) Generation() int32 { return s.generation }

// TestConsumerParentReplacement waits for old children without stopping a peer.
func TestConsumerParentReplacement(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		group := NewRoot(nil)
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
		time.Sleep(2 * time.Second)
		synctest.Wait()
		if ref.Borrow().Generation() != 1 || args.starts.Load() != 1 {
			t.Fatal("parent replacement began before old child cleanup finished")
		}
		peer.checkRunning(t)
		close(finish)
		<-oldWorker.stopped
		newWorker := <-args.workers
		<-newWorker.started
		if run := <-args.runs; run != 2 || ref.Borrow().Generation() != 2 {
			t.Fatal("parent replacement did not publish and run")
		}
		synctest.Wait()
		peer.checkRunning(t)
		<-group.Stop()
		<-newWorker.stopped
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
func (s *consumerRouter) Init() error {
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
		models := StartGroup(group)
		oldVersion := StartGroup(models)
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
		challenger := StartGroup(models)
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

package service

import (
	"context"
	"sync"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// PoolArgs are the arguments of a WorkerPool.
type PoolArgs struct {
	im.Shared
	Workers int
}

// WorkerArgs are the arguments of a Worker.
type WorkerArgs struct {
	im.Shared
	ID int
}

// Worker counts the jobs that it does.
type Worker struct {
	supervise.Service[WorkerArgs]
	mu   sync.Mutex
	done int
}

// Run waits until ctx ends.
func (w *Worker) Run(ctx context.Context) { <-ctx.Done() }

// Do records one job and returns the ID of the worker.
func (w *Worker) Do() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.done++
	return w.Args().ID
}

// WorkerPool starts its workers under the group of its run.
type WorkerPool struct {
	supervise.Service[PoolArgs]
}

// Run starts the workers and a nested group, and waits until ctx ends.
func (p *WorkerPool) Run(ctx context.Context) {
	sup := p.Children()
	for i := range p.Args().Workers {
		if _, err := supervise.Start[Worker](sup, WorkerArgs{ID: i}); err != nil {
			panic(err)
		}
	}
	spare := supervise.Reserve[Worker]()
	if err := supervise.StartReserved(p.Children(), spare, WorkerArgs{ID: -1}); err != nil {
		panic(err)
	}
	if _, err := supervise.Start[Worker](supervise.StartGroup((p.Children())), WorkerArgs{ID: -2}); err != nil {
		panic(err)
	}
	<-ctx.Done()
}

// Count records entries through an ordinary synchronous callback.
func (w *Worker) Count(entries *sync.Map) {
	w.mu.Lock()
	defer w.mu.Unlock()
	entries.Range(func(_, _ any) bool { w.done++; return true })
}

// add starts a child from a closure that keeps the parent group local.
func (p *WorkerPool) add() {
	children := p.Children()
	start := func() { supervise.Start[Worker](children, WorkerArgs{}) }
	start()
}

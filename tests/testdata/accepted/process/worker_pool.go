package process

import (
	"context"
	"sync"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
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
	proc.Process[WorkerArgs]
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

// WorkerPool starts its workers under the supervisor of its run.
type WorkerPool struct {
	proc.Process[PoolArgs]
}

// Run starts the workers and a nested group, and waits until ctx ends.
func (p *WorkerPool) Run(ctx context.Context) {
	sup := p.Supervisor()
	for i := range p.Args().Workers {
		proc.Start[Worker](sup, WorkerArgs{ID: i})
	}
	spare := proc.Reserve[Worker]()
	proc.StartReserved(p.Supervisor(), spare, WorkerArgs{ID: -1})
	proc.Start[Worker](proc.StartSupervisor((p.Supervisor())), WorkerArgs{ID: -2})
	<-ctx.Done()
}

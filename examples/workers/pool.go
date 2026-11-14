package workers

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// PoolArgs sets the worker count and the conversion version for a pool.
type PoolArgs struct {
	im.Shared
	Workers int
	Version int
}

// Pool distributes text jobs among the workers owned by its current run.
type Pool struct {
	supervise.Service[PoolArgs]
	mu      sync.Mutex
	next    int
	workers []supervise.Ref[Worker]
	ready   chan struct{}
	stopped chan struct{}
}

// Init creates the workers before callers can use the pool.
func (p *Pool) Init() error {
	if p.Args().Workers < 1 {
		return errors.New("worker count must be positive")
	}
	p.ready = make(chan struct{})
	p.stopped = make(chan struct{})
	children := p.Children()
	for id := range p.Args().Workers {
		worker, err := supervise.Start[Worker](children, WorkerArgs{ID: id, Version: p.Args().Version})
		if err != nil {
			return err
		}
		p.workers = append(p.workers, worker)
	}
	return nil
}

// Run publishes readiness after every child can receive work.
func (p *Pool) Run(ctx context.Context) {
	defer close(p.stopped)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for !p.workersReady() {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
	close(p.ready)
	<-ctx.Done()
}

// workersReady reports whether every child has started its job loop.
func (p *Pool) workersReady() bool {
	for _, worker := range p.workers {
		if !worker.Current().Ready() {
			return false
		}
	}
	return true
}

// Ready reports whether every worker can receive jobs.
func (p *Pool) Ready() bool {
	select {
	case <-p.stopped:
		return false
	default:
	}
	select {
	case <-p.ready:
		return true
	default:
		return false
	}
}

// Process sends a job to the next worker without holding the pool's mutex.
func (p *Pool) Process(text string) Result {
	if !p.Ready() {
		return Result{}
	}
	p.mu.Lock()
	worker := p.workers[p.next]
	p.next = (p.next + 1) % len(p.workers)
	p.mu.Unlock()
	return worker.Current().Process(text)
}

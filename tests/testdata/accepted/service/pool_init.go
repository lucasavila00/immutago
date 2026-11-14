package service

import (
	"context"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Pool owns the requested number of workers.
type Pool struct {
	supervise.Service[int]
}

// Init creates the workers before the pool becomes available.
func (p *Pool) Init() error {
	children := p.Children()
	for id := range p.Args() {
		if _, err := supervise.Start[Worker](children, WorkerArgs{ID: id}); err != nil {
			return err
		}
	}
	return nil
}

// Run keeps the pool active until shutdown.
func (*Pool) Run(ctx context.Context) { <-ctx.Done() }

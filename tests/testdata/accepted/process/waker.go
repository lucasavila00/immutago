package process

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/proc"
)

// Waker runs a step each time a caller wakes it.
type Waker struct {
	proc.Process[proc.None]
	wake  chan struct{}
	steps int
}

// Init prepares the channel before callers can reach the process.
func (w *Waker) Init() { w.wake = make(chan struct{}, 1) }

// Run counts one step after each wake until ctx ends.
func (w *Waker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.wake:
			w.steps++
		}
	}
}

// Wake asks for one more step.
func (w *Waker) Wake() {
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

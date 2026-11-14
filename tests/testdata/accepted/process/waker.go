package process

import (
	"context"
	"sync"

	"github.com/lucasavila00/immutago/pkg/proc"
)

// Waker runs a step each time a caller wakes it.
type Waker struct {
	proc.Process[proc.None]
	mu    sync.Mutex
	wake  chan struct{}
	steps int
}

// wakeChannel creates the channel once, before either Run or Wake uses it.
func (w *Waker) wakeChannel() chan struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.wake == nil {
		w.wake = make(chan struct{}, 1)
	}
	return w.wake
}

// Run counts one step after each wake until ctx ends.
func (w *Waker) Run(ctx context.Context) {
	wake := w.wakeChannel()
	for {
		select {
		case <-ctx.Done():
			return
		case <-wake:
			w.steps++
		}
	}
}

// Wake requests one step, including before Run begins. Repeated requests can combine into one step.
func (w *Waker) Wake() {
	wake := w.wakeChannel()
	select {
	case wake <- struct{}{}:
	default:
	}
}

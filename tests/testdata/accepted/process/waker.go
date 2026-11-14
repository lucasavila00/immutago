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

// Init creates the channel before callers can use Wake.
func (w *Waker) Init() error {
	w.wake = make(chan struct{}, 1)
	return nil
}

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

// Wake requests one step, including before Run begins. Repeated requests can combine into one step.
func (w *Waker) Wake() {
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

// StartWaker forwards a locally declared startup result pair.
func StartWaker(root *proc.Supervisor) (proc.PID[Waker], error) {
	var pid, err = proc.Start[Waker](root, proc.None{})
	return pid, err
}

// StartAssignedWaker binds startup results to existing local variables.
func StartAssignedWaker(root *proc.Supervisor) (proc.PID[Waker], error) {
	var pid proc.PID[Waker]
	var err error
	(pid), (err) = (proc.Start[Waker](root, proc.None{}))
	return pid, err
}

// StartDiscardedWaker discards both results in a local declaration.
func StartDiscardedWaker(root *proc.Supervisor) {
	var _, _ = proc.Start[Waker](root, proc.None{})
}

package service

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Waker runs a step each time a caller wakes it.
type Waker struct {
	supervise.Service[supervise.NoArgs]
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
func StartWaker(root *supervise.Group) (supervise.Ref[Waker], error) {
	var ref, err = supervise.Start[Waker](root, supervise.NoArgs{})
	return ref, err
}

// StartAssignedWaker binds startup results to existing local variables.
func StartAssignedWaker(root *supervise.Group) (supervise.Ref[Waker], error) {
	var ref supervise.Ref[Waker]
	var err error
	(ref), (err) = (supervise.Start[Waker](root, supervise.NoArgs{}))
	return ref, err
}

// StartDiscardedWaker discards both results in a local declaration.
func StartDiscardedWaker(root *supervise.Group) {
	var _, _ = supervise.Start[Waker](root, supervise.NoArgs{})
}

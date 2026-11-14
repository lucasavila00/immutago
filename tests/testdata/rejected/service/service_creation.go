package service

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Ticker is a service.
type Ticker struct {
	supervise.Service[supervise.NoArgs]
	ticks int
}

// Run waits until ctx ends.
func (t *Ticker) Run(ctx context.Context) { <-ctx.Done() }

// Ticks returns the tick count.
func (t *Ticker) Ticks() int { return t.ticks }

// Plain is not a service: a pointer embedding does not mark it.
type Plain struct {
	*supervise.Service[supervise.NoArgs]
}

// Run returns at once.
func (*Plain) Run(context.Context) {}

// Twin is a defined copy of a service type.
type Twin Ticker

// Create makes services without supervise.Start.
func Create(root *supervise.Group) {
	literal := &Ticker{}
	var zero Ticker
	allocated := new(Ticker)
	ticks := (*Ticker).Ticks
	supervise.Start[Plain](root, supervise.NoArgs{})
	var scalar supervise.Ref[int]
	reserved := supervise.Reserve[int]()
	_, _, _, _, _, _ = literal, zero, allocated, ticks, scalar, reserved
}

package process

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/proc"
)

// Ticker is a process.
type Ticker struct {
	proc.Process[proc.None]
	ticks int
}

// Run waits until ctx ends.
func (t *Ticker) Run(ctx context.Context) { <-ctx.Done() }

// Ticks returns the tick count.
func (t *Ticker) Ticks() int { return t.ticks }

// Plain is not a process: a pointer embedding does not mark it.
type Plain struct{ *proc.Process[proc.None] }

// Run returns at once.
func (*Plain) Run(context.Context) {}

// Twin is a defined copy of a process type.
type Twin Ticker

// Create makes processes without proc.Start.
func Create(root *proc.Supervisor) {
	literal := &Ticker{}
	var zero Ticker
	allocated := new(Ticker)
	ticks := (*Ticker).Ticks
	proc.Start[Plain](root, proc.None{})
	var scalar proc.PID[int]
	reserved := proc.Reserve[int]()
	_, _, _, _, _, _ = literal, zero, allocated, ticks, scalar, reserved
}

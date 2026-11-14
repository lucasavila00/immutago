package process

import (
	"context"
	"sync"

	"github.com/lucasavila00/immutago/pkg/proc"
)

// Status is a defined scalar result.
type Status int

// Refresher keeps mutable state.
type Refresher struct {
	proc.Process
	mu    sync.Mutex
	table map[string]int
}

// Run waits until ctx ends.
func (r *Refresher) Run(ctx context.Context) { <-ctx.Done() }

// Table returns mutable storage.
func (r *Refresher) Table() map[string]int { return r.table }

// Each passes a callback.
func (r *Refresher) Each(visit func(string)) {}

// Load returns an interface.
func (r *Refresher) Load() error { return nil }

// Keys takes a variadic slice.
func (r *Refresher) Keys(keys ...string) Status { return Status(len(keys)) }

// Self returns the process pointer.
func (r *Refresher) Self() *Refresher { return r }

// Value uses a value receiver.
func (r Refresher) Value() Status { return 0 }

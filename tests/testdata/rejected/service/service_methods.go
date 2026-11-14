package service

import (
	"context"
	"sync"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Status is a defined scalar result.
type Status int

// Refresher keeps mutable state.
type Refresher struct {
	supervise.Service[supervise.NoArgs]
	mu    sync.Mutex
	table map[string]int
}

// Run waits until ctx ends.
func (r *Refresher) Run(ctx context.Context) { <-ctx.Done() }

// Table returns mutable storage.
func (r *Refresher) Table() map[string]int { return r.table }

// Load returns an interface.
func (r *Refresher) Load() error { return nil }

// Self returns the service pointer.
func (r *Refresher) Self() *Refresher { return r }

// Value uses a value receiver.
func (r Refresher) Value() Status { return 0 }

package api

import (
	"context"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Counter uses scalar arguments, so its contracts contain no im type.
type Counter struct{ supervise.Service[int] }

// Run waits for cancellation.
func (*Counter) Run(ctx context.Context) { <-ctx.Done() }

// Count reads the start argument.
func (p *Counter) Count() int { return p.Args() }

// Handle is a Ref alias.
type Handle = supervise.Ref[Counter]

var current Handle

// Get returns the current Ref.
func Get() Handle { return current }

// Contract keeps a Ref in a generic alias constraint.
type Contract[T interface{ ~int | Handle }] = int

// ReadyCounter initializes its count before methods can read it.
type ReadyCounter struct {
	supervise.Service[int]
	count int
}

// Init sets the initial count.
func (p *ReadyCounter) Init() error { p.count = p.Args(); return nil }

// Run waits for cancellation.
func (*ReadyCounter) Run(ctx context.Context) { <-ctx.Done() }

// ReadyHandle is a Ref for a service with an explicit Init method.
type ReadyHandle = supervise.Ref[ReadyCounter]

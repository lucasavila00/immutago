package api

import (
	"context"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Counter uses scalar arguments, so its contracts contain no im type.
type Counter struct{ proc.Process[int] }

// Run waits for cancellation.
func (*Counter) Run(ctx context.Context) { <-ctx.Done() }

// Count reads the start argument.
func (p *Counter) Count() int { return p.Args() }

// Handle is a PID alias.
type Handle = proc.PID[Counter]

var current Handle

// Get returns the current PID.
func Get() Handle { return current }

// Contract keeps a PID in a generic alias constraint.
type Contract[T interface{ ~int | Handle }] = int

// ReadyCounter initializes its count before methods can read it.
type ReadyCounter struct {
	proc.Process[int]
	count int
}

// Init sets the initial count.
func (p *ReadyCounter) Init() error { p.count = p.Args(); return nil }

// Run waits for cancellation.
func (*ReadyCounter) Run(ctx context.Context) { <-ctx.Done() }

// ReadyHandle is a PID for a process with an explicit Init method.
type ReadyHandle = proc.PID[ReadyCounter]

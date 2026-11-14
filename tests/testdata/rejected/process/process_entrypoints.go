package process

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/proc"
)

// MissingRun has no process entry point.
type MissingRun struct{ proc.Process[proc.None] }

// WrongRun has an entry point with a different parameter type.
type WrongRun struct{ proc.Process[proc.None] }

// Run cannot receive the runtime context.
func (*WrongRun) Run(int) {}

// ResultRun returns a value from the entry point.
type ResultRun struct{ proc.Process[proc.None] }

// Run has an unsupported result.
func (*ResultRun) Run(context.Context) int { return 0 }

// Runner has a Run method but is not a process.
type Runner struct{}

// Run waits for cancellation.
func (*Runner) Run(ctx context.Context) { <-ctx.Done() }

// PromotedRun inherits Run instead of declaring a process receiver.
type PromotedRun struct {
	proc.Process[proc.None]
	Runner
}

// WrongInit has an initializer with a result.
type WrongInit struct{ proc.Process[proc.None] }

// Run waits for cancellation.
func (*WrongInit) Run(ctx context.Context) { <-ctx.Done() }

// Init has an unsupported result.
func (*WrongInit) Init() int { return 0 }

// ValueInit has an initializer with a value receiver.
type ValueInit struct{ proc.Process[proc.None] }

// Run waits for cancellation.
func (*ValueInit) Run(ctx context.Context) { <-ctx.Done() }

// Init cannot change the process value.
func (ValueInit) Init() {}

// Initializer has an Init method but is not a process.
type Initializer struct{}

// Init does nothing.
func (*Initializer) Init() {}

// PromotedInit inherits Init instead of declaring a process receiver.
type PromotedInit struct {
	proc.Process[proc.None]
	Initializer
}

// Run waits for cancellation.
func (*PromotedInit) Run(ctx context.Context) { <-ctx.Done() }

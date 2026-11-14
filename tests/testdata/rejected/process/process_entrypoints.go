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

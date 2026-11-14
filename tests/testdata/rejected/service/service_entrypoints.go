package service

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// MissingRun has no service entry point.
type MissingRun struct {
	supervise.Service[supervise.NoArgs]
}

// WrongRun has an entry point with a different parameter type.
type WrongRun struct {
	supervise.Service[supervise.NoArgs]
}

// Run cannot receive the runtime context.
func (*WrongRun) Run(int) {}

// ResultRun returns a value from the entry point.
type ResultRun struct {
	supervise.Service[supervise.NoArgs]
}

// Run has an unsupported result.
func (*ResultRun) Run(context.Context) int { return 0 }

// Runner has a Run method but is not a service.
type Runner struct{}

// Run waits for cancellation.
func (*Runner) Run(ctx context.Context) { <-ctx.Done() }

// PromotedRun inherits Run instead of declaring a service receiver.
type PromotedRun struct {
	supervise.Service[supervise.NoArgs]
	Runner
}

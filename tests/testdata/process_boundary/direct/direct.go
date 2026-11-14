package direct

import (
	"context"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Counter is a process without any im contract.
type Counter struct{ proc.Process[int] }

// Run waits for cancellation.
func (*Counter) Run(ctx context.Context) { <-ctx.Done() }

// saved creates a process outside Start.
var saved = new(Counter)

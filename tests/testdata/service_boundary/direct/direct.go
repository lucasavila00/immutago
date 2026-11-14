package direct

import (
	"context"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Counter is a service without any im contract.
type Counter struct{ supervise.Service[int] }

// Run waits for cancellation.
func (*Counter) Run(ctx context.Context) { <-ctx.Done() }

// saved creates a service outside Start.
var saved = new(Counter)

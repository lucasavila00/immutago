package direct

import (
	"context"
	"sync/atomic"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Counter is a service without any im contract.
type Counter struct {
	supervise.Service[supervise.NoArgs]
	paused atomic.Bool
	step   atomic.Int64
}

// Pause stops counting.
func (c *Counter) Pause() { c.paused.Store(true) }

// SetStep sets the increment for each count.
func (c *Counter) SetStep(step int64) { c.step.Store(step) }

// Run waits for cancellation.
func (*Counter) Run(ctx context.Context) { <-ctx.Done() }

// saved creates a service outside Start.
var saved = new(Counter)

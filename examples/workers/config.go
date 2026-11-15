package workers

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// VariantSet is one immutable snapshot of the requested variant names.
type VariantSet struct {
	im.Frozen
	Names im.FrozenSlice[string]
}

// VariantConfig keeps configuration outside the dispatcher that consumes it.
type VariantConfig struct {
	supervise.Service[supervise.NoArgs]
	current im.Cell[VariantSet]
}

// Run keeps the configuration service available until shutdown.
func (*VariantConfig) Run(ctx context.Context) { <-ctx.Done() }

// Set publishes the requested names.
func (c *VariantConfig) Set(names im.FrozenSlice[string]) {
	c.current.Store(VariantSet{Names: names})
}

// Snapshot returns a local view of the current variant set.
func (c *VariantConfig) Snapshot() im.View[VariantSet] {
	return c.current.Load()
}

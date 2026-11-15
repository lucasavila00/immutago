package workers

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// VariantSet is one immutable snapshot of the requested variant names.
type VariantSet struct {
	im.Scoped
	Names im.FrozenSlice[im.FrozenString]
}

// VariantConfig keeps configuration outside the dispatcher that consumes it.
type VariantConfig struct {
	supervise.Service[supervise.NoArgs]
	current im.Owner[VariantSet]
}

// Run keeps the configuration service available until shutdown.
func (*VariantConfig) Run(ctx context.Context) { <-ctx.Done() }

// Set publishes the requested names.
func (c *VariantConfig) Set(names im.FrozenSlice[im.FrozenString]) {
	c.current.Store(&VariantSet{Names: names})
}

// Snapshot returns the current variant set, or its zero value before the first Set.
func (c *VariantConfig) Snapshot() VariantSet {
	current := c.current.Load()
	if current == nil {
		return VariantSet{}
	}
	return *current
}

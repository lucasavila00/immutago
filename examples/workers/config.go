package workers

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// VariantSet is one immutable snapshot of the requested variant names.
type VariantSet struct {
	im.Scoped
	Names im.FrozenSlice[string]
}

// VariantConfig keeps configuration outside the dispatcher that consumes it.
type VariantConfig struct {
	supervise.Service[supervise.NoArgs]
	current im.Owner[VariantSet]
}

// Run keeps the configuration service available until shutdown.
func (*VariantConfig) Run(ctx context.Context) { <-ctx.Done() }

// Set publishes the requested names and returns their count.
func (c *VariantConfig) Set(names im.FrozenSlice[string]) int {
	c.current.Store(&VariantSet{Names: names})
	return names.Len()
}

// Names reads the current names, or an empty collection before the first Set.
func (c *VariantConfig) Names() im.FrozenSlice[string] {
	current := c.current.Load()
	if current == nil {
		var none im.FrozenSlice[string]
		return none
	}
	return current.Names
}

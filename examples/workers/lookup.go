package workers

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// PoolRoute gives callers a reference to a selected pool when one is available.
type PoolRoute struct {
	im.Local
	Available bool
	Pool      supervise.Ref[Pool]
}

// Lookup returns the current pool for a variant without exposing dispatcher state.
func (d *Dispatcher) Lookup(name string) PoolRoute {
	d.mu.Lock()
	defer d.mu.Unlock()
	if current := d.groups[name].current; current != nil {
		return PoolRoute{Available: true, Pool: current.pool}
	}
	return PoolRoute{}
}

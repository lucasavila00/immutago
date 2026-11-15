package workers

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// DispatcherArgs selects the configuration, pool size, and polling interval.
type DispatcherArgs struct {
	Config  supervise.Ref[VariantConfig]
	Workers int
	Every   time.Duration
}

// versionGroup owns the workers of one text conversion version.
type versionGroup struct {
	group   *supervise.Group
	pool    supervise.Ref[Pool]
	version int
}

// variant holds the serving version and a replacement that is starting.
type variant struct {
	current *versionGroup
	pending *versionGroup
}

// Dispatcher keeps child pools in step with the configured variant names.
type Dispatcher struct {
	supervise.Service[DispatcherArgs]
	changes sync.Mutex
	mu      sync.Mutex
	running atomic.Bool
	groups  map[string]variant
}

// Init validates the polling settings and creates the group index.
func (d *Dispatcher) Init() error {
	if d.Args().Every <= 0 || d.Args().Workers < 1 {
		return errors.New("polling interval and worker count must be positive")
	}
	d.groups = make(map[string]variant)
	return nil
}

// Run applies configuration changes and publishes ready versions until shutdown.
func (d *Dispatcher) Run(ctx context.Context) {
	d.running.Store(true)
	defer d.running.Store(false)
	ticker := time.NewTicker(d.Args().Every)
	defer ticker.Stop()
	for {
		d.Refresh()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Refresh starts new variants, stops removed variants, and publishes ready versions.
func (d *Dispatcher) Refresh() int {
	d.changes.Lock()
	defer d.changes.Unlock()
	if !d.running.Load() {
		return 0
	}
	wanted := make(map[string]bool)
	projection := d.Args().Config.Current().Variants()
	var names im.SliceProjection[string]
	if !projection.IsNil() {
		names = projection.Names()
	}
	for _, item := range names.All() {
		name := im.Text(item).Clone()
		wanted[name] = true
		d.mu.Lock()
		_, exists := d.groups[name]
		if !exists {
			d.groups[name] = variant{}
		}
		d.mu.Unlock()
		if !exists {
			d.prepare(name, 1)
		}
	}
	d.remove(wanted)
	d.publishReady()
	d.mu.Lock()
	count := len(d.groups)
	d.mu.Unlock()
	return count
}

// Load stages a replacement for an existing variant without changing its serving pool.
func (d *Dispatcher) Load(name string, version int) bool {
	d.changes.Lock()
	defer d.changes.Unlock()
	if !d.running.Load() {
		return false
	}
	d.mu.Lock()
	current, exists := d.groups[name]
	d.mu.Unlock()
	if !exists {
		return false
	}
	if current.current != nil && current.current.version == version ||
		current.pending != nil && current.pending.version == version {
		return true
	}
	return d.prepare(name, version)
}

// prepare starts a candidate and stops any older candidate. The caller holds changes.
func (d *Dispatcher) prepare(name string, version int) bool {
	group := supervise.StartGroup(d.Children())
	pool, err := supervise.Start[Pool](group, PoolArgs{Workers: d.Args().Workers, Version: version})
	if err != nil {
		<-group.Stop()
		return false
	}
	d.mu.Lock()
	current := d.groups[name]
	previous := current.pending
	current.pending = &versionGroup{group: group, pool: pool, version: version}
	d.groups[name] = current
	d.mu.Unlock()
	if previous != nil {
		<-previous.group.Stop()
	}
	return true
}

// remove stops every group whose name was removed. The caller holds changes.
func (d *Dispatcher) remove(wanted map[string]bool) {
	var removed []*supervise.Group
	d.mu.Lock()
	for name, current := range d.groups {
		if wanted[name] {
			continue
		}
		delete(d.groups, name)
		if current.current != nil {
			removed = append(removed, current.current.group)
		}
		if current.pending != nil {
			removed = append(removed, current.pending.group)
		}
	}
	d.mu.Unlock()
	for _, group := range removed {
		<-group.Stop()
	}
}

// publishReady switches to started pools before stopping old groups. The caller holds changes.
func (d *Dispatcher) publishReady() {
	var previous []*supervise.Group
	d.mu.Lock()
	for name, current := range d.groups {
		if current.pending == nil || !current.pending.pool.Current().Ready() {
			continue
		}
		if current.current != nil {
			previous = append(previous, current.current.group)
		}
		current.current = current.pending
		current.pending = nil
		d.groups[name] = current
	}
	d.mu.Unlock()
	for _, group := range previous {
		<-group.Stop()
	}
}

// Version returns a variant's serving version, or zero while no pool is ready.
func (d *Dispatcher) Version(name string) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	if current := d.groups[name].current; current != nil {
		return current.version
	}
	return 0
}

// Process selects the serving pool without holding a lock while the job runs.
func (d *Dispatcher) Process(name, text string) Result {
	d.mu.Lock()
	current := d.groups[name].current
	d.mu.Unlock()
	if current == nil {
		return Result{}
	}
	return current.pool.Current().Process(text)
}

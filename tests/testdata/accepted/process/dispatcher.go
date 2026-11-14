package process

import (
	"context"
	"sync"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// VariantSet is one published set of variant names.
type VariantSet struct {
	im.Scoped
	Names im.FrozenSlice[string]
}

// VariantConfig keeps the current set of variants. It survives a restart of the Dispatcher.
type VariantConfig struct {
	proc.Process[proc.None]
	current im.Owner[VariantSet]
}

// Run waits until ctx ends.
func (*VariantConfig) Run(ctx context.Context) { <-ctx.Done() }

// Set publishes a new set of variant names.
func (c *VariantConfig) Set(names im.FrozenSlice[string]) { c.current.Store(&VariantSet{Names: names}) }

// Names returns the current variant names, or no names before the first Set.
func (c *VariantConfig) Names() im.FrozenSlice[string] {
	set := c.current.Load()
	if set == nil {
		var none im.FrozenSlice[string]
		return none
	}
	return set.Names
}

// GroupArgs are the arguments of a Throttle.
type GroupArgs struct {
	im.Shared
	Variant string
	Rate    int
}

// Throttle limits the requests of one variant.
type Throttle struct{ proc.Process[GroupArgs] }

// Run waits until ctx ends.
func (*Throttle) Run(ctx context.Context) { <-ctx.Done() }

// VersionLoaderArgs are the arguments of a VersionLoader.
type VersionLoaderArgs struct {
	im.Shared
	Variant string
	Self    proc.PID[VersionLoader]
}

// VersionArgs are the arguments of a VersionServer.
type VersionArgs struct {
	im.Shared
	Version int
	Loader  proc.PID[VersionLoader]
}

// VersionServer serves one version of a variant.
type VersionServer struct{ proc.Process[VersionArgs] }

// Run tells the loader that the version runs, then waits until ctx ends.
func (s *VersionServer) Run(ctx context.Context) {
	s.Args().Loader.Proc().Ready(s.Args().Version)
	<-ctx.Done()
}

// VersionLoader starts one VersionServer for each loaded version, each in its own nested supervisor.
type VersionLoader struct {
	proc.Process[VersionLoaderArgs]
	mu      sync.Mutex
	pending map[int]*proc.Supervisor
	current *proc.Supervisor
	version int
}

// Run waits until ctx ends.
func (*VersionLoader) Run(ctx context.Context) { <-ctx.Done() }

// Load starts a server for version. Repeated requests for a pending or current version do nothing.
// The server calls Ready when it runs.
func (l *VersionLoader) Load(version int) {
	l.mu.Lock()
	_, pending := l.pending[version]
	if pending || l.current != nil && l.version == version {
		l.mu.Unlock()
		return
	}
	next := proc.StartSupervisor(l.Supervisor())
	if l.pending == nil {
		l.pending = make(map[int]*proc.Supervisor)
	}
	l.pending[version] = next
	l.mu.Unlock()
	proc.Start[VersionServer](next, VersionArgs{Version: version, Loader: l.Args().Self})
}

// Ready makes a running version current and stops the server of the previous version.
func (l *VersionLoader) Ready(version int) {
	l.mu.Lock()
	next, ok := l.pending[version]
	delete(l.pending, version)
	previous := l.current
	if ok {
		l.current, l.version = next, version
	}
	l.mu.Unlock()
	if ok && previous != nil {
		<-previous.Stop()
	}
}

// Version returns the version whose server runs, or 0 before the first server runs.
func (l *VersionLoader) Version() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.version
}

// DispatcherArgs are the arguments of a Dispatcher.
type DispatcherArgs struct {
	im.Shared
	Config proc.PID[VariantConfig]
}

// group is the nested supervisor and the loader of one variant.
type group struct {
	sub    *proc.Supervisor
	loader proc.PID[VersionLoader]
}

// Dispatcher keeps one group of processes for each configured variant.
type Dispatcher struct {
	proc.Process[DispatcherArgs]
	mu     sync.Mutex
	groups map[string]group
}

// Run starts the groups of the current variants, because a restart starts with no groups, and waits until ctx ends.
func (d *Dispatcher) Run(ctx context.Context) {
	d.Refresh()
	<-ctx.Done()
}

// Refresh reads the current configuration, starts new groups, stops removed groups, and returns the group count.
func (d *Dispatcher) Refresh() int {
	d.mu.Lock()
	wanted := make(map[string]bool)
	for _, name := range d.Args().Config.Proc().Names().All() {
		wanted[name] = true
	}
	var removed []*proc.Supervisor
	if d.groups == nil {
		d.groups = make(map[string]group)
	}
	for name := range wanted {
		if _, ok := d.groups[name]; !ok {
			sub := proc.StartSupervisor(d.Supervisor())
			proc.Start[Throttle](sub, GroupArgs{Variant: name, Rate: 10})
			loader := proc.Reserve[VersionLoader]()
			proc.StartReserved(sub, loader, VersionLoaderArgs{Variant: name, Self: loader})
			d.groups[name] = group{sub: sub, loader: loader}
		}
	}
	for name, g := range d.groups {
		if !wanted[name] {
			removed = append(removed, g.sub)
			delete(d.groups, name)
		}
	}
	count := len(d.groups)
	d.mu.Unlock()
	for _, sub := range removed {
		<-sub.Stop()
	}
	return count
}

// Load asks the loader of one variant to load a version, and stops the group of the variant if the version is
// not valid. It returns false if the variant has no group.
func (d *Dispatcher) Load(variant string, version int) bool {
	d.mu.Lock()
	g, ok := d.groups[variant]
	if ok && version < 0 {
		delete(d.groups, variant)
	}
	d.mu.Unlock()
	switch {
	case !ok:
		return false
	case version < 0:
		<-g.sub.Stop()
		return false
	}
	g.loader.Proc().Load(version)
	return true
}

// useDispatcher starts two variant groups, requests one version, then removes one group.
func useDispatcher(root *proc.Supervisor) {
	config := proc.Start[VariantConfig](root, proc.None{})
	initial := im.FreezeSlice([]string{"blue", "green"})
	config.Proc().Set(initial)
	dispatcher := proc.Start[Dispatcher](root, DispatcherArgs{Config: config})
	dispatcher.Proc().Refresh()
	dispatcher.Proc().Load("blue", 1)

	next := im.FreezeSlice([]string{"green"})
	config.Proc().Set(next)
	dispatcher.Proc().Refresh()
}

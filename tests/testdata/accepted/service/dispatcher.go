package service

import (
	"context"
	"sync"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// VariantSet is one published set of variant names.
type VariantSet struct {
	im.Scoped
	Names im.FrozenSlice[string]
}

// VariantConfig keeps the current set of variants. It survives a restart of the Dispatcher.
type VariantConfig struct {
	supervise.Service[supervise.NoArgs]
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
type Throttle struct{ supervise.Service[GroupArgs] }

// Run waits until ctx ends.
func (*Throttle) Run(ctx context.Context) { <-ctx.Done() }

// VersionLoaderArgs are the arguments of a VersionLoader.
type VersionLoaderArgs struct {
	im.Shared
	Variant string
	Self    supervise.Ref[VersionLoader]
}

// VersionArgs are the arguments of a VersionServer.
type VersionArgs struct {
	im.Shared
	Version int
	Loader  supervise.Ref[VersionLoader]
}

// VersionServer serves one version of a variant.
type VersionServer struct{ supervise.Service[VersionArgs] }

// Run tells the loader that the version runs, then waits until ctx ends.
func (s *VersionServer) Run(ctx context.Context) {
	s.Args().Loader.Current().Ready(s.Args().Version)
	<-ctx.Done()
}

// VersionLoader starts one VersionServer for each loaded version, each in its own nested group.
type VersionLoader struct {
	supervise.Service[VersionLoaderArgs]
	mu      sync.Mutex
	pending map[int]*supervise.Group
	current *supervise.Group
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
	next := supervise.StartGroup(l.Children())
	if l.pending == nil {
		l.pending = make(map[int]*supervise.Group)
	}
	l.pending[version] = next
	l.mu.Unlock()
	if _, err := supervise.Start[VersionServer](next, VersionArgs{Version: version, Loader: l.Args().Self}); err != nil {
		panic(err)
	}
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
	Config supervise.Ref[VariantConfig]
}

// group is the nested group and the loader of one variant.
type group struct {
	sub    *supervise.Group
	loader supervise.Ref[VersionLoader]
}

// Dispatcher keeps one group of services for each configured variant.
type Dispatcher struct {
	supervise.Service[DispatcherArgs]
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
	for _, name := range d.Args().Config.Current().Names().All() {
		wanted[name] = true
	}
	var removed []*supervise.Group
	if d.groups == nil {
		d.groups = make(map[string]group)
	}
	for name := range wanted {
		if _, ok := d.groups[name]; !ok {
			sub := supervise.StartGroup(d.Children())
			if _, err := supervise.Start[Throttle](sub, GroupArgs{Variant: name, Rate: 10}); err != nil {
				panic(err)
			}
			loader := supervise.Reserve[VersionLoader]()
			if err := supervise.StartReserved(sub, loader, VersionLoaderArgs{Variant: name, Self: loader}); err != nil {
				panic(err)
			}
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
	g.loader.Current().Load(version)
	return true
}

// useDispatcher starts two variant groups, requests one version, then removes one group.
func useDispatcher(root *supervise.Group) {
	config, err := supervise.Start[VariantConfig](root, supervise.NoArgs{})
	if err != nil {
		panic(err)
	}
	initial := im.FreezeSlice([]string{"blue", "green"})
	config.Current().Set(initial)
	dispatcher, err := supervise.Start[Dispatcher](root, DispatcherArgs{Config: config})
	if err != nil {
		panic(err)
	}
	dispatcher.Current().Refresh()
	dispatcher.Current().Load("blue", 1)

	next := im.FreezeSlice([]string{"green"})
	config.Current().Set(next)
	dispatcher.Current().Refresh()
}

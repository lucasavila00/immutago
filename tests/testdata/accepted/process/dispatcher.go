package process

import (
	"context"
	"sync"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// GroupArgs are the arguments of the processes in the group of one variant.
type GroupArgs struct {
	im.Shared
	Variant string
	Rate    int
}

// Throttle limits the requests of one variant.
type Throttle struct{ proc.Process[GroupArgs] }

// Run waits until ctx ends.
func (*Throttle) Run(ctx context.Context) { <-ctx.Done() }

// VersionArgs are the arguments of a VersionServer.
type VersionArgs struct {
	im.Shared
	Variant string
	Version int
}

// VersionServer serves one version of a variant.
type VersionServer struct{ proc.Process[VersionArgs] }

// Run waits until ctx ends.
func (*VersionServer) Run(ctx context.Context) { <-ctx.Done() }

// Version returns the served version.
func (s *VersionServer) Version() int { return s.Args().Version }

// VersionLoader starts one VersionServer for each loaded version.
type VersionLoader struct {
	proc.Process[GroupArgs]
	mu      sync.Mutex
	current *proc.Supervisor
}

// Run waits until ctx ends.
func (*VersionLoader) Run(ctx context.Context) { <-ctx.Done() }

// Load starts a server for version, then stops the server of the previous version.
func (l *VersionLoader) Load(version int) int {
	next := proc.StartSupervisor(l.Supervisor())
	server := proc.Start[VersionServer](next, VersionArgs{Variant: l.Args().Variant, Version: version})
	ready := server.Proc().Version()
	l.mu.Lock()
	previous := l.current
	l.current = next
	l.mu.Unlock()
	if previous != nil {
		<-previous.Stop()
	}
	return ready
}

// group is the nested supervisor and the loader of one variant.
type group struct {
	sub    *proc.Supervisor
	loader proc.PID[VersionLoader]
}

// Dispatcher keeps one group of processes for each configured variant.
type Dispatcher struct {
	proc.Process[proc.None]
	mu     sync.Mutex
	groups map[string]group
}

// Run waits until ctx ends.
func (*Dispatcher) Run(ctx context.Context) { <-ctx.Done() }

// Apply starts the groups of new variants, stops the groups of removed variants, and returns the group count.
func (d *Dispatcher) Apply(variants im.FrozenSlice[string]) int {
	wanted := make(map[string]bool)
	for _, name := range variants.All() {
		wanted[name] = true
	}
	var removed []*proc.Supervisor
	d.mu.Lock()
	if d.groups == nil {
		d.groups = make(map[string]group)
	}
	for name := range wanted {
		if _, ok := d.groups[name]; !ok {
			sub := proc.StartSupervisor(d.Supervisor())
			args := GroupArgs{Variant: name, Rate: 10}
			proc.Start[Throttle](sub, args)
			d.groups[name] = group{sub: sub, loader: proc.Start[VersionLoader](sub, args)}
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

// Load loads a version for one variant and stops the group of the variant if the version is not valid.
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
	return g.loader.Proc().Load(version) == version
}

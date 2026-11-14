package process

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Counter writes a map that it gives to a Reporter.
type Counter struct {
	proc.Process[proc.None]
	seen map[string]int
}

// Run starts a Reporter with the map, then writes the map.
// proc.Trust is a false promise here: the checker accepts the program, but it has a data race.
func (c *Counter) Run(ctx context.Context) {
	c.seen = map[string]int{}
	children := proc.NewRoot(nil)
	defer func() { <-children.Stop() }()
	proc.Start[Reporter](children, ReporterArgs{Seen: proc.Trust(c.seen)})
	for i := 0; ctx.Err() == nil; i++ {
		c.seen["k"] = i
	}
}

// ReporterArgs are the arguments of a Reporter.
type ReporterArgs struct {
	im.Shared
	Seen proc.UncheckedDependency[map[string]int]
}

// Reporter reads the map of a Counter.
type Reporter struct {
	proc.Process[ReporterArgs]
	total int
}

// Run reads the map until ctx ends.
func (r *Reporter) Run(ctx context.Context) {
	for ctx.Err() == nil {
		r.total += r.Args().Seen.Value()["k"]
	}
}

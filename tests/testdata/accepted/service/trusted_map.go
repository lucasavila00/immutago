package service

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Counter writes a map that it gives to a Reporter.
type Counter struct {
	supervise.Service[supervise.NoArgs]
	seen map[string]int
}

// Run starts a Reporter with the map, then writes the map.
// supervise.AssumeConcurrentSafe is a false promise here: the checker accepts the program, but it has a data race.
func (c *Counter) Run(ctx context.Context) {
	c.seen = map[string]int{}
	children := supervise.NewRoot(nil)
	defer func() { <-children.Stop() }()
	if _, err := supervise.Start[Reporter](children, ReporterArgs{Seen: supervise.AssumeConcurrentSafe(c.seen)}); err != nil {
		panic(err)
	}
	for i := 0; ctx.Err() == nil; i++ {
		c.seen["k"] = i
	}
}

// ReporterArgs are the arguments of a Reporter.
type ReporterArgs struct {
	im.Shared
	Seen supervise.ConcurrentSafe[map[string]int]
}

// Reporter reads the map of a Counter.
type Reporter struct {
	supervise.Service[ReporterArgs]
	total int
}

// Run reads the map until ctx ends.
func (r *Reporter) Run(ctx context.Context) {
	for ctx.Err() == nil {
		r.total += r.Args().Seen.Value()["k"]
	}
}

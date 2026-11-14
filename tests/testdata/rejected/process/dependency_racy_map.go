package process

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/proc"
)

// Tally gives its own map to an Auditor without proc.Trust.
type Tally struct {
	proc.Process[proc.None]
	seen map[string]int
}

// Run starts an Auditor with the map, then writes the map.
func (t *Tally) Run(ctx context.Context) {
	t.seen = map[string]int{}
	children := proc.NewRoot(nil)
	defer func() { <-children.Stop() }()
	proc.Start[Auditor](children, AuditorArgs{Seen: t.seen})
	for i := 0; ctx.Err() == nil; i++ {
		t.seen["k"] = i
	}
}

// AuditorArgs have a plain map.
type AuditorArgs struct {
	im.Shared
	Seen map[string]int
}

// Auditor reads the map of a Tally.
type Auditor struct {
	proc.Process[AuditorArgs]
	total int
}

// Run reads the map until ctx ends.
func (a *Auditor) Run(ctx context.Context) {
	for ctx.Err() == nil {
		a.total += a.Args().Seen["k"]
	}
}

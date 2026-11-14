package process

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/proc"
)

// Counter keeps a mutable count.
type Counter struct {
	proc.Process[proc.None]
	count int
}

// Run waits until ctx ends.
func (c *Counter) Run(ctx context.Context) { <-ctx.Done() }

// Count returns the count.
func (c *Counter) Count() int { return c.count }

// add increases the count.
func (c *Counter) add() { c.count++ }

var saved *Counter

// use accepts any value.
func use(any) {}

// Transport moves a process pointer from Proc() in each rejected form.
func Transport(pid proc.PID[Counter]) {
	process := pid.Proc()
	saved = pid.Proc()
	_ = pid.Proc().count
	pid.Proc().add()
	pid.Proc().Run(context.Background())
	count := pid.Proc().Count
	get := pid.Proc
	go pid.Proc().Count()
	defer pid.Proc().Count()
	use(pid.Proc())
	copied := *pid.Proc()
	_ = pid.Proc().Args()
	_, _, _, _ = process, count, get, copied
}

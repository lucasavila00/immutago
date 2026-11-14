package service

import (
	"context"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// Counter keeps a mutable count.
type Counter struct {
	supervise.Service[supervise.NoArgs]
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

// Transport moves a service pointer from Current() in each rejected form.
func Transport(ref supervise.Ref[Counter]) {
	service := ref.Current()
	saved = ref.Current()
	_ = ref.Current().count
	ref.Current().add()
	ref.Current().Run(context.Background())
	count := ref.Current().Count
	get := ref.Current
	go ref.Current().Count()
	defer ref.Current().Count()
	use(ref.Current())
	copied := *ref.Current()
	_ = ref.Current().Args()
	_, _, _, _ = service, count, get, copied
}

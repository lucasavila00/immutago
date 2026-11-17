package rejected

import (
	"context"
	"sync"

	"example.com/consumer/hooks"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Counter struct {
	supervise.Service[chan int]
	mu    *sync.Mutex
	limit int
}

func (c *Counter) Init(*supervise.Supervisor) error { c.mu = new(sync.Mutex); return nil }
func (c *Counter) SetLimit(n int)                   { c.mu.Lock(); c.limit = n; c.mu.Unlock() }
func (c Counter) Run(context.Context) {
	c.mu.Lock()
	n := c.limit
	c.mu.Unlock()
	c.Args() <- n
}

type ValueInit struct {
	supervise.Service[int]
	limit int
}

func (w ValueInit) Init(*supervise.Supervisor) error { w.limit = 5; return nil }
func (*ValueInit) Run(context.Context)               {}

type PromotedValue struct {
	supervise.Service[int]
	hooks.ValueRun
}

type PromotedPointer struct {
	supervise.Service[int]
	*hooks.ValueRun
}

type Base struct{ supervise.Service[int] }
type PromotedValueInit struct {
	Base
	hooks.ValueInit
}

func (*PromotedValueInit) Run(context.Context) {}

type PromotedPointerInit struct {
	Base
	*hooks.ValueInit
}

func (*PromotedPointerInit) Run(context.Context) {}

type InterfaceRun struct {
	supervise.Service[int]
	hooks.Runner
}

type InterfaceInit struct {
	Base
	hooks.Initializer
}

func (*InterfaceInit) Run(context.Context) {}

type Alias = Counter
type Defined Counter

func (Defined) Run(context.Context) {}

type Anonymous = struct {
	supervise.Service[int]
	hooks.ValueRun
}

type _ struct {
	supervise.Service[int]
	hooks.ValueRun
}
type _ = Counter

func Start(root *supervise.Supervisor) {
	_, _ = supervise.Start[Counter](root, make(chan int, 1))
	_, _ = supervise.Start[Alias](root, make(chan int, 1))
	_, _ = supervise.Start[Defined](root, make(chan int, 1))
	_, _ = supervise.Start[Anonymous](root, 1)
	_, _ = supervise.Start[struct {
		supervise.Service[int]
		hooks.ValueRun
	}](root, 1)
	type Local struct {
		supervise.Service[int]
		hooks.ValueRun
	}
	_, _ = supervise.Start[Local](root, 1)
}

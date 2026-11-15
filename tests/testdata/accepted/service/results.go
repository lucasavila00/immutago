package service

import (
	"context"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Reply struct {
	im.Scoped
	Count int
}
type SharedReply struct {
	im.Shared
	Count int
}
type Counter struct {
	supervise.Service[map[string]int]
	count   int
	current im.Owner[Reply]
}

func (*Counter) Run(context.Context) {}
func (c *Counter) Count() int        { return c.count }
func (c *Counter) Read() Reply       { return Reply{Count: c.count} }
func (*Counter) Array() [2]int       { return [2]int{1, 2} }

type Listing struct {
	im.Scoped
	Values im.FrozenSlice[int]
}
type Table struct {
	im.Scoped
	Values im.FrozenMap[int, int]
}

func (*Counter) List() Listing { frozen := im.FreezeSlice([]int{1, 2}); return Listing{Values: frozen} }
func (*Counter) Table() Table {
	frozen := im.FreezeMap(map[int]int{1: 1})
	return Table{Values: frozen}
}
func (c *Counter) Input(callback func(int), values map[string]int) int {
	callback(c.count)
	values["count"] = c.count
	return c.count
}
func (c *Counter) local() *int { return &c.count }
func (c *Counter) Capture() int {
	f := func() { c.count++; c.current.Store(&Reply{Count: c.count}) }
	f()
	return c.count
}
func use(ref supervise.Ref[Counter]) {
	c := ref.Current()
	c.count++
	f := c.Count
	_ = f()
	go c.Count()
	defer c.Count()
	children := c.Children()
	other := children
	_ = other
}

// User embedding and promoted methods use the same result contract.
type Other struct{ *Counter }
type Alias = Counter

// Immutable values and Shared pointers can be stored as start arguments.
type StoredValue struct{ supervise.Service[Reply] }
type StoredShared struct {
	supervise.Service[*SharedReply]
}

func (*Counter) Ack() im.Scoped { return im.Scoped{} }

func (c *Counter) Update() { c.count++ }

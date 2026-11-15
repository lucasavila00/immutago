package service

import (
	"context"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Reply struct {
	im.ByValue
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
	im.ByValue
	Values im.FrozenSlice[int]
}
type Table struct {
	im.ByValue
	Values im.FrozenMap[string, int]
}

func (*Counter) List() Listing { frozen := im.FreezeSlice([]int{1, 2}); return Listing{Values: frozen} }
func (*Counter) Table() Table {
	frozen := im.FreezeMap(map[string]int{"a": 1})
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
	ref.Current().Update()
	_ = ref.Current().Count()
	go ref.Current().Count()
	defer ref.Current().Count()
}

// User embedding and promoted methods use the same result contract.
type Other struct{ *Counter }
type Alias = Counter

// Immutable values and Shared pointers can be stored as start arguments.
type StoredValue struct{ supervise.Service[Reply] }
type StoredShared struct {
	supervise.Service[*SharedReply]
}

func (*Counter) Ack() im.ByValue { return im.ByValue{} }

func (c *Counter) Update() { c.count++ }

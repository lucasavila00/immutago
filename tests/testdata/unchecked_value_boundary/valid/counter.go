package valid

import (
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
)

// CounterHolder uses ordinary replaceable storage for an opaque handle.
type CounterHolder struct{ Counter api.Counter }

// CounterReply uses the same handle rule in a Local record.
type CounterReply struct {
	im.Local
	Counter api.Counter
}

// Counter returns a user-defined handle through the ordinary result rule.
func (*Loader) Counter() api.Counter { return api.NewCounter() }

// CounterResult combines that handle with an ordinary value and error.
func (l *Loader) CounterResult() (CounterReply, int, error) {
	return CounterReply{Counter: l.Counter()}, 1, nil
}

// KeepCounter stores, copies, and captures the handle without a service-specific path.
func KeepCounter(value api.Counter) func() int64 {
	holder := CounterHolder{Counter: value}
	holder.Counter = value
	return func() int64 { return holder.Counter.Add(1) }
}

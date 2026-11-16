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

// CounterSnapshot reads stored handles without generating a projection for each handle.
type CounterSnapshot struct {
	im.Frozen
	Counter api.Counter
	Pointer *api.Counter
}

// ReadCounterSnapshot returns the same persistent handle from both generated getters.
func ReadCounterSnapshot(snapshot CounterSnapshotProjection) (api.Counter, *api.Counter) {
	return snapshot.Counter(), snapshot.Pointer()
}

// EmbeddedCounter permits promoted value methods.
type EmbeddedCounter struct{ api.Counter }

// EmbeddedCounterPointer permits promoted pointer methods on stored pointers.
type EmbeddedCounterPointer struct{ *api.Counter }

// ReadEmbeddedCounter copies the wrapper for a value receiver call.
func ReadEmbeddedCounter(value EmbeddedCounter) int64 { return value.Add(1) }

// KeepEmbeddedCounter uses an existing pointer without taking an embedded field address.
func KeepEmbeddedCounter(value EmbeddedCounterPointer) *api.Counter { return value.Pointer() }

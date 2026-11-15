//go:generate go run github.com/lucasavila00/immutago/cmd/immutago-gen

package valid

import (
	"context"
	"example.com/consumer/api"
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
	"sync/atomic"
)

// Model keeps a trusted handle in immutable storage.
type Model struct {
	im.Frozen
	Table  im.UncheckedValue[api.Table]
	Models [2][1]Handle
}

// Wrapper embeds an ordinary handle without exposing the payload.
type Wrapper struct{ im.UncheckedValue[api.Table] }

// Copy keeps ordinary handle transport available outside the declaring package.
func Copy() (Model, any) {
	value := api.New()
	return Model{Table: value}, Wrapper{UncheckedValue: value}
}

// Read reaches the payload through its declaring package.
func Read(model *Model) int { return api.Read(model.Table) }

// FreezeSlice stores an unchecked value without inspecting its payload.
func FreezeSlice() im.FrozenSlice[im.UncheckedValue[api.Table]] {
	values := []im.UncheckedValue[api.Table]{api.New()}
	frozen := im.FreezeSlice(values)
	return frozen
}

// FreezeMap permits the same handle as a frozen map element.
func FreezeMap() im.FrozenMap[string, im.UncheckedValue[api.Table]] {
	values := map[string]im.UncheckedValue[api.Table]{"table": api.New()}
	frozen := im.FreezeMap(values)
	return frozen
}

// Handle keeps the library identity through an alias.
type Handle = im.UncheckedValue[api.Table]

// Serving combines handles with local request data.
type Serving struct {
	im.Local
	Model   Handle
	Models  [2]Handle
	Version int
}

// Loader publishes a current trusted model.
type Loader struct {
	supervise.Service[supervise.NoArgs]
	current atomic.Pointer[Handle]
}

// Init publishes the first handle.
func (l *Loader) Init() error { value := api.New(); l.current.Store(&value); return nil }

// Run waits for shutdown.
func (*Loader) Run(ctx context.Context) { <-ctx.Done() }

// Current returns the handle without exposing its payload.
func (l *Loader) Current() Handle { return *l.current.Load() }

// Read returns an optional ordinary error with the handle.
func (l *Loader) Read() (Handle, error) { return l.Current(), nil }

// Many returns an unnamed array through the same read-result rule.
func (l *Loader) Many() [2]Handle { return [2]Handle{l.Current()} }

// Serving returns request data with a nested handle array.
func (l *Loader) Serving() Serving { return Serving{Model: l.Current(), Models: l.Many(), Version: 1} }

// ReadProjection reads direct and nested array getters.
func ReadProjection(value ModelProjection) int {
	return api.Read(value.Table()) + api.Read(value.Models().At(0).At(0))
}

// ReadElements extracts handles from slice and map projections.
func ReadElements(values im.SliceProjection[Handle], entries im.MapProjection[int, Handle]) int {
	first := im.Value(values.At(0))
	next, ok := entries.Get(1)
	if ok {
		return api.Read(first) + api.Read(im.Value(next))
	}
	return 0
}

// RawHolder keeps the Cell in a direct named field.
type RawHolder struct{ Cell im.Cell[Handle] }

// ReadCell uses the same Frozen and read-result proofs as collection reads.
func ReadCell(holder *RawHolder) Handle { return im.Value(holder.Cell.Load()) }

// CopyArray reads zero and nested unnamed arrays of handles.
func CopyArray(value im.Projection[[2][1]Handle]) [2][1]Handle { return im.Value(value) }

// ZeroArray reads a zero-length array without changing its category.
func ZeroArray(value im.Projection[[0]Handle]) [0]Handle { return im.Value(value) }

// Capture extracts the ordinary handle before the closure.
func Capture(value Serving) func() Handle {
	handle := value.Model
	return func() Handle { return handle }
}

// CaptureProjection extracts the handle before capturing it.
func CaptureProjection(value ModelProjection) func() Handle {
	handle := value.Table()
	return func() Handle { return handle }
}

// Ordinary keeps interface, generic, callback, go, and defer transport available.
func Ordinary(value Handle) any {
	f := func() Handle { return value }
	go discard(value)
	defer discard(value)
	return identity(f())
}

// identity copies an ordinary value.
func identity[T any](value T) T { return value }

// discard accepts an ordinary handle.
func discard(Handle) {}

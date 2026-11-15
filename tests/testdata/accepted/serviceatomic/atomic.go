package serviceatomic

import (
	"context"
	atom "sync/atomic"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Worker struct {
	supervise.Service[supervise.NoArgs]
}

func (*Worker) Run(context.Context) {}
func (*Worker) Count() int          { return 1 }

type WorkerRef = supervise.Ref[Worker]
type Pointer[T any] = atom.Pointer[T]
type Selected = Pointer[WorkerRef]
type Slot struct{ Ref WorkerRef }
type SlotAlias = Slot

type Dispatcher struct {
	Direct  atom.Pointer[WorkerRef]
	Wrapped Pointer[SlotAlias]
}

func Replace(d *Dispatcher, ref WorkerRef) int {
	d.Direct.Store(&ref)
	loaded := d.Direct.Load()
	previous := d.Direct.Swap(loaded)
	d.Direct.CompareAndSwap(previous, &ref)
	d.Wrapped.Store(&Slot{Ref: ref})
	slot := d.Wrapped.Load()
	d.Wrapped.Swap(slot)
	d.Wrapped.CompareAndSwap(slot, nil)
	d.Wrapped.Store(slot)
	return loaded.Current().Count() + d.Wrapped.Load().Ref.Current().Count()
}

func Empty() *WorkerRef {
	selected := new(Selected)
	selected.Store(nil)
	return selected.Load()
}

func Zero() Selected { return Selected{} }

func Callback(ref WorkerRef) int {
	var current atom.Pointer[func() WorkerRef]
	produce := func() WorkerRef { return ref }
	current.Store(&produce)
	return (*current.Load())().Current().Count()
}

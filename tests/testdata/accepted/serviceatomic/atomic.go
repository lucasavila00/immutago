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
type Forwarded[T any] = Pointer[T]
type Selected = Pointer[WorkerRef]
type Slot struct{ Ref WorkerRef }
type SlotAlias = Slot

type Dispatcher struct {
	Direct  atom.Pointer[WorkerRef]
	Wrapped Forwarded[SlotAlias]
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
	return loaded.Borrow().Count() + d.Wrapped.Load().Ref.Borrow().Count()
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
	return (*current.Load())().Borrow().Count()
}

// Interfaces permits the ordinary interface operations of atomic storage.
func Interfaces(ref WorkerRef) any {
	var value atom.Value
	value.Store(ref)
	var pointer atom.Pointer[WorkerRef]
	pointer.Store(&ref)
	var hidden any = pointer.Load()
	return hidden
}

// GenericStorage uses the same Go rules as other generic containers.
func GenericStorage(ref WorkerRef) {
	var nested atom.Pointer[Box[WorkerRef]]
	nested.Store(&Box[WorkerRef]{Value: ref})
	value := identity(nested.Load())
	value.Value.Borrow().Count()
	var ignored Ignored[WorkerRef]
	var defined Defined[WorkerRef]
	_, _ = ignored, defined
}

type Box[T any] struct{ Value T }
type Ignored[T any] = atom.Pointer[WorkerRef]
type Defined[T any] atom.Pointer[T]

// identity returns an ordinary Go value.
func identity[T any](value T) T { return value }

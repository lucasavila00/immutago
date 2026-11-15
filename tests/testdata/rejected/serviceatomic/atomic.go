package serviceatomic

import (
	"context"
	"sync/atomic"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Worker struct {
	supervise.Service[supervise.NoArgs]
}

func (*Worker) Run(context.Context) {}
func (*Worker) Count() int          { return 1 }

type WorkerRef = supervise.Ref[Worker]
type Box[T any] struct{ Value T }
type Nested[T any] = atomic.Pointer[Box[T]]
type Ignored[T any] = atomic.Pointer[WorkerRef]
type Pointer[T any] struct{ Value T }
type Defined[T any] atomic.Pointer[T]
type LocalSlot struct {
	im.Local
	Ref WorkerRef
}
type MixedSlot struct {
	Ref   WorkerRef
	Local LocalSlot
}
type ProtectedSlot struct {
	im.Local
	Current atomic.Pointer[WorkerRef]
}
type FrozenSlot struct {
	Ref    WorkerRef
	Values im.FrozenSlice[int]
}

var user Box[WorkerRef]
var nested Nested[WorkerRef]
var ignored Ignored[WorkerRef]
var lookalike Pointer[WorkerRef]
var defined Defined[WorkerRef]
var local atomic.Pointer[LocalSlot]
var mixed atomic.Pointer[MixedSlot]
var frozen atomic.Pointer[FrozenSlot]

func Erase(ref WorkerRef) {
	var value atomic.Value
	value.Store(ref)
	var pointer atomic.Pointer[WorkerRef]
	pointer.Store(&ref)
	var hidden any = pointer.Load()
	_ = hidden
}

func Extract(pointer *atomic.Pointer[WorkerRef]) {
	service := pointer.Load().Current()
	_ = service
}

func Generic[T any](value T) T { return value }

func Transport(pointer *atomic.Pointer[WorkerRef]) {
	_ = Generic(pointer)
}

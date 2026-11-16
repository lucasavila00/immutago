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

var local atomic.Pointer[LocalSlot]
var mixed atomic.Pointer[MixedSlot]

func Extract(pointer *atomic.Pointer[WorkerRef]) {
	service := pointer.Load().Borrow()
	_ = service
}

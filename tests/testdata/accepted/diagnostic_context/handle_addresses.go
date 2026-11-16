package diagnosticcontext

import (
	"sync/atomic"

	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type Router struct {
	current atomic.Pointer[supervise.Ref[Worker]]
	backup  supervise.Ref[Worker]
}

func (r *Router) Select(ref supervise.Ref[Worker], all []supervise.Ref[Worker]) {
	r.current.Store(new(ref))
	r.current.Store(new(r.backup))
	r.current.Store(new(all[0]))
	_ = &supervise.Ref[Worker]{}
	_ = new(supervise.Ref[Worker])
}

type PointerRouter struct{ current atomic.Pointer[Worker] }

func (r *PointerRouter) Select(ref supervise.Ref[Worker]) { r.current.Store(ref.Borrow()) }

type counter struct{ N int }

type Counter struct {
	im.Frozen
	state im.UncheckedValue[counter]
}

type HandleHolder struct{ Handle Counter }

func wrapperAddresses(handle Counter, holder HandleHolder, handles []Counter) {
	_ = new(handle)
	_ = new(holder.Handle)
	_ = new(handles[0])
	_ = &Counter{}
	_ = new(Counter)
}

type RawHolder struct{ Raw im.UncheckedValue[counter] }

func rawAddresses(raw im.UncheckedValue[counter], holder RawHolder, values []im.UncheckedValue[counter]) {
	_ = &raw
	_ = &holder.Raw
	_ = &values[0]
}

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
	r.current.Store(&ref)
	r.current.Store(&r.backup)
	r.current.Store(&all[0])
}

type counter struct{ N int }

type Counter struct {
	im.Frozen
	state im.UncheckedValue[counter]
}

type HandleHolder struct{ Handle Counter }

func wrapperAddresses(handle Counter, holder HandleHolder, handles []Counter) {
	_ = &handle
	_ = &holder.Handle
	_ = &handles[0]
}

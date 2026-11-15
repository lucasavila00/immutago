package service

import (
	"github.com/lucasavila00/immutago/pkg/im"
	"github.com/lucasavila00/immutago/pkg/supervise"
)

type optionalRoute struct{ cache supervise.Ref[Counter] }

func optionalCount(r optionalRoute) int {
	if !r.cache.IsZero() {
		return r.cache.Borrow().Count()
	}
	return 0
}

func localOptionalCount(route Route) int {
	if !route.Worker.IsZero() {
		return route.Worker.Borrow().Count()
	}
	return 0
}

type CounterRef = supervise.Ref[Counter]
type embeddedRef struct{ CounterRef }

type EmbeddedRoute struct {
	im.Local
	CounterRef
}

func explicitEmbeddedZero(route EmbeddedRoute) bool { return route.CounterRef.IsZero() }

type FrozenRoute struct {
	im.Frozen
	Worker  CounterRef
	Workers [1][2]CounterRef
}

func frozenZero(route *FrozenRoute) bool {
	return ((route.Worker).IsZero)() && route.Workers[0][1].IsZero()
}

func projectedZero(route FrozenRouteProjection) bool {
	return route.Worker().IsZero() && route.Workers().At(0).At(1).IsZero()
}

func zeroForms(ref CounterRef, refs map[string]CounterRef, p *CounterRef, holder embeddedRef) bool {
	var zero CounterRef
	check := ref.IsZero
	method := CounterRef.IsZero
	var optional interface{ IsZero() bool } = ref
	callback := func() bool { return ref.IsZero() }
	go ref.IsZero()
	defer ref.IsZero()
	return zero.IsZero() && check() && method(ref) && optional.IsZero() &&
		callback() && refs["cache"].IsZero() && p.IsZero() && holder.IsZero()
}

func localZeroCallback(route Route) func() bool {
	ref := route.Worker
	return func() bool { return ref.IsZero() }
}

func localZeroMethod(route Route) func() bool {
	ref := route.Worker
	go ref.IsZero()
	defer ref.IsZero()
	return ref.IsZero
}

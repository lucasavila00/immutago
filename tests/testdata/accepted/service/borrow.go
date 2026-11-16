package service

import "github.com/lucasavila00/immutago/pkg/supervise"

var savedWorker *Counter

func saveWorker(ref supervise.Ref[Counter]) *Counter {
	worker := ref.Borrow()
	savedWorker = worker
	worker.Count()
	worker.Count()
	return worker
}

func frozenWorker(route FrozenRoute) *Counter              { return route.Worker.Borrow() }
func localWorker(route Route) *Counter                     { return route.Worker.Borrow() }
func projectedWorker(route FrozenRouteProjection) *Counter { return route.Worker().Borrow() }

func borrowMethods(ref CounterRef, wrapped embeddedRef, refs map[string]CounterRef) func() int {
	accessor := ref.Borrow
	method := CounterRef.Borrow
	_ = accessor()
	_ = method(ref)
	_ = wrapped.Borrow()
	_ = refs["worker"].Borrow()
	return ref.Borrow().Count
}

func savedCallback(route Route) func() int {
	worker := route.Worker.Borrow()
	return func() int { return worker.Count() }
}

func localBorrowMethod(route Route) func() *Counter        { return route.Worker.Borrow }
func frozenBorrowMethod(route FrozenRoute) func() *Counter { return route.Worker.Borrow }

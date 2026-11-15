package servicecross

import "example.com/consumer/serviceatomic"

func AtomicCount(ref serviceatomic.WorkerRef) int {
	var current serviceatomic.Pointer[serviceatomic.WorkerRef]
	current.Store(&ref)
	return current.Load().Borrow().Count()
}

package valid

import "example.com/consumer/api"

type promoted struct{ api.Handle }

// promotedCall uses a promoted Ref accessor.
func promotedCall(ref promoted) { ref.Borrow().Count() }

// saveAccessor keeps the producer ordinary and uses its pointer locally.
func saveAccessor(ref api.Handle) { current := ref.Borrow; worker := current(); worker.Count() }

type Pointer = *api.Alias

func readPointer(worker Pointer) int { return worker.Count() }

var packageValue api.Worker
var next = api.Pointer
var method = api.WorkerRef.Borrow

type Source struct{ Next func() *api.Worker }

// PointerBindings uses aliases, assignments, tuple results, and exact direct parameters.
func PointerBindings(ref api.WorkerRef) bool {
	worker := ref.Borrow()
	other := worker
	var zero Pointer
	worker = api.Current
	worker = api.Pointer()
	worker, number := api.Pointers()
	var selected, count = api.Pointers()
	selected, count = api.Pointers()
	_, _ = worker.Pair()
	_ = zero
	_ = readPointer(worker) + readPointer(ref.Borrow()) + selected.Count() + number + count
	return other != nil && worker == selected
}

// PointerSources checks construction and producer contracts without a direct Borrow chain.
func PointerSources(ref api.WorkerRef, anonymous api.AnonymousRef) int {
	value := api.Worker{}
	worker := &value
	worker = &api.Worker{Value: 1}
	worker = new(api.Worker)
	worker = new(value)
	worker = new(api.Worker{})
	holder := struct{ Value api.Worker }{}
	worker = &holder.Value
	array := [1]api.Worker{}
	worker = &array[0]
	values := []api.Worker{{}}
	worker = &values[0]
	worker = &packageValue
	worker = api.New[api.Worker]()
	worker = api.KeepPointer(worker)
	producer := api.Keep(api.Pointer)
	worker = producer()
	borrow := api.Keep(ref.Borrow)
	worker = borrow()
	var selected interface{ Borrow() *api.Worker } = ref
	worker = selected.Borrow()
	worker = api.WorkerRef.Borrow(ref)
	worker = (interface{ Borrow() *api.Worker }).Borrow(selected)
	worker = method(ref)
	worker = (Source{Next: next}).Next()
	local := anonymous.Borrow()
	return worker.Promoted() + local.Promoted()
}

type DefinedRef api.WorkerRef
type ReferenceAlias = DefinedRef
type ReferenceHolder struct{ Values [1]ReferenceAlias }

// ReferenceCopies keeps defined references ordinary across every containing type.
func ReferenceCopies(ref api.WorkerRef) api.WorkerRef {
	holders := []ReferenceHolder{{[1]ReferenceAlias{DefinedRef(ref)}}}
	copyRef := func() ReferenceAlias { return api.Keep(holders[0].Values[0]) }
	var boxed any = api.Keep(copyRef)()
	result := api.WorkerRef(boxed.(ReferenceAlias))
	result = api.WorkerRef(api.DefinedRef(result))
	return result
}

// ServiceValues keeps ordinary service values outside the Local pointer rule.
func ServiceValues(value api.Worker) any {
	copyValue := api.Keep(value)
	change := func() { copyValue.Value++ }
	change()
	values := []api.Worker{copyValue}
	return api.Box[api.Worker]{Value: values[0]}
}

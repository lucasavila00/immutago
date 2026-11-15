package valid

import "example.com/consumer/api"

type borrower interface{ Borrow() *api.Counter }

type holder struct{ worker api.Handle }

// OnFailure supplies the library callback without reading the reference.
func (*holder) OnFailure(error) {}

type embedded struct{ api.Handle }
type nested struct{ embedded }

// Interfaces checks interface transport of references, holders, and promoted methods.
func Interfaces(ref api.Handle) any {
	var zero api.Handle
	var nilHolder *holder
	_ = any(zero)
	_ = any(nilHolder)
	var value any = ref
	value = &holder{worker: ref}
	var sink interface{ OnFailure(error) } = &holder{worker: ref}
	sink.OnFailure(nil)
	_ = interface{ OnFailure(error) }(&holder{worker: ref})
	value = holder{worker: ref}
	value = any(embedded{ref})
	value = nested{embedded{ref}}
	var selected borrower = ref
	selected = borrower(embedded{ref})
	selected = nested{embedded{ref}}
	pointer := selected.Borrow()
	_ = pointer.Args()
	_ = borrower.Borrow(selected)
	accessor := selected.Borrow
	_ = accessor()
	_ = borrower(ref)
	return value
}

// Recovery checks type assertions and switches for references and nested values.
func Recovery(value any) api.Handle {
	if ref, ok := value.(api.Handle); ok {
		ref.Borrow().Count()
	}
	if wrapped, ok := value.(holder); ok {
		wrapped.worker.Borrow().Count()
	}
	if wrapped, ok := value.(nested); ok {
		var selected borrower = wrapped
		_ = selected.Borrow()
	}
	switch selected := value.(type) {
	case api.Handle:
		selected.Borrow().Count()
	case *holder:
		selected.worker.Borrow().Count()
	case borrower:
		_ = selected.Borrow()
	}
	return value.(api.Handle)
}

// consume accepts ordinary interface arguments, including variadic arguments.
func consume(any, ...any) {}

// pair supplies a tuple whose first result contains a reference.
func pair(ref api.Handle) (api.Handle, int) { return ref, 1 }

// identity checks ordinary generic calls after interface conversion.
func identity[T any](value T) T { return value }

// Arguments checks explicit conversions, tuple arguments, and variadic elements.
func Arguments(ref api.Handle) {
	_ = identity(any(ref))
	consume(ref, holder{ref}, embedded{ref}, nested{embedded{ref}})
	consume(pair(ref))
	consume(ref, []any{ref, holder{ref}}...)
	_ = struct{ Value any }{ref}
	_ = map[any]any{ref: holder{ref}}
	_ = []any{ref, embedded{ref}}
}

// NativeTransport checks interface destinations in ordinary container operations.
func NativeTransport(ref api.Handle, channel chan any) {
	channel <- ref
	values := []api.Handle{ref}
	var value any
	for _, value = range values {
		_ = value
	}
	for value = range func(yield func(api.Handle) bool) { yield(ref) } {
		_ = value
	}
	_, value = pair(ref)
	value, _ = pair(ref)
	_ = ref == value
	switch value {
	case ref:
	}
	switch ref {
	case value:
	}
}

type producer interface{ Worker() api.Handle }

// Contracts checks interface changes for callbacks and interface method contracts.
func Contracts(ref api.Handle, source producer) {
	var value any = func() api.Handle { return ref }
	value = source
	_ = value.(producer)
	_ = value.(func() api.Handle)
	var other interface{ Worker() api.Handle } = source
	other.Worker().Borrow().Count()
}

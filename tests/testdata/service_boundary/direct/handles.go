package direct

import "example.com/consumer/api"

func saveAccessor(ref api.Handle)                                          { current := ref.Current; _ = current() }
func replace(ref api.Handle)                                               { *ref.Current() = api.Counter{} }
func box(ref api.Handle) any                                               { return ref }
func indirect[T interface{ Current() *api.Counter }](value T) *api.Counter { return value.Current() }
func generic(ref api.Handle)                                               { _ = indirect(ref) }

type Copy api.Handle

func convertBack(value Copy)           { ref := api.Handle(value); ref.Current().Count() }
func eraseCallback(ref api.Handle) any { return func() api.Handle { return ref } }
func iteratorBox(ref api.Handle) {
	values := func(yield func(api.Handle) bool) { yield(ref) }
	var value any
	for value = range values {
		_ = value
	}
}

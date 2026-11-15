package direct

import "example.com/consumer/api"

func saveAccessor(ref api.Handle)                                         { current := ref.Borrow; _ = current() }
func replace(ref api.Handle)                                              { *ref.Borrow() = api.Counter{} }
func indirect[T interface{ Borrow() *api.Counter }](value T) *api.Counter { return value.Borrow() }
func generic(ref api.Handle)                                              { _ = indirect(ref) }

type Copy api.Handle

func convertBack(value Copy) { ref := api.Handle(value); ref.Borrow().Count() }

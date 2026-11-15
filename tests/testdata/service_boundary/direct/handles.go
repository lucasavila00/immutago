package direct

import "example.com/consumer/api"

func saveAccessor(ref api.Handle)                                          { current := ref.Current; _ = current() }
func replace(ref api.Handle)                                               { *ref.Current() = api.Counter{} }
func box(ref api.Handle) any                                               { return ref }
func indirect[T interface{ Current() *api.Counter }](value T) *api.Counter { return value.Current() }
func generic(ref api.Handle)                                               { _ = indirect(ref) }

package direct

import "example.com/consumer/api"

func saveAccessor(ref api.Handle) { current := ref.Borrow; _ = current() }
func replace(ref api.Handle)      { *ref.Borrow() = api.Counter{} }

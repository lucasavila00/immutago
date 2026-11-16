package direct

import "example.com/consumer/api"

func replace(ref api.Handle) { *ref.Borrow() = api.Counter{} }

package contracts

import "example.com/consumer/api"

func retain(*api.Worker)

func call(ref api.WorkerRef) { retain(ref.Borrow()) }

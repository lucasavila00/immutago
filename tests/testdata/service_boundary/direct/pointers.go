package direct

import "example.com/consumer/api"

func fields(value *api.Worker) {
	_ = value.Value
	value.Value++
	_ = *value
	*value = api.Worker{}
	_ = value.Count
	_ = (*api.Worker).Count(value)
	_ = value.Init()
	value.Run(nil)
	_ = value.Args()
	_ = value.Children()
	_ = value.Service
}

func genericReturn[T interface{ Borrow() *api.Worker }](ref T) *api.Worker {
	return ref.Borrow()
}

func returnAnonymous(ref api.AnonymousRef) *api.Anonymous { return ref.Borrow() }

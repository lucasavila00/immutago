package rejectedconsumer

import "example.com/consumer/api"

func calls() {
	value := api.Product{}
	pointer := &api.Product{}
	_ = value.ReadCost()
	_ = pointer.Cost()
	method := pointer.ReadCost
	_ = method
}

package consumer

import "example.com/consumer/api"

func Cost() int {
	value := api.Product{Price: 8}
	pointer := &api.Product{Price: 10}
	settings := &api.Settings{Limit: 100}
	return value.Cost() + pointer.ReadCost() + settings.Remaining(30)
}

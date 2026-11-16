package consumer

import "example.com/consumer/api"

func number(n int) int { return n }

func GenericReads() int {
	value := api.Product{Price: 3}
	pointer := &api.Product{Price: 3}
	read := value.Map[int]
	return value.Map(number) + value.Map[int](number) + pointer.MapPointer(number) + read(number)
}

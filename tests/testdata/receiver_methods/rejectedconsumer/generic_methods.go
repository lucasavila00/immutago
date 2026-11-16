package rejectedconsumer

import "example.com/consumer/api"

func local(n int) api.Settings { return api.Settings{Limit: n} }
func number(n int) int         { return n }

func genericCalls(value api.Product) {
	_ = value.Map[api.Settings](local)
	_ = value.MapPointer[int](number)
}
